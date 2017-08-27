package host

import (
	"circlet/messages"
	"fmt"
	"github.com/golang/protobuf/proto"
	"log"
	"math/rand"
	"net"
	"time"
)

// raftChannel is probably poorly named. most of the things in here are pertaining to raft
type RaftChannels struct {
	heartbeat       chan *messages.HeartBeat // receives heartbeats
	electionTimeout chan uint64              // periodic fixed interval to check for election timeouts. messages are a timestamp close to the current time
	holdElection    chan *messages.HoldElection
	numPeers        chan int
}

// at some point may want to multiplex on a channel instead of select over multiple
// go's select over multiple channels relies on the futex syscall which
// be called more as more channels are being monitored
func NewRaftChannels() *RaftChannels {
	return &RaftChannels{
		heartbeat:       make(chan *messages.HeartBeat, 32),
		electionTimeout: make(chan uint64, 32),
		holdElection:    make(chan *messages.HoldElection, 32),
		numPeers:        make(chan int, 4),
	}
}

func (rx *Server) raftLoop() {
	//state
	var (
		lastHeartbeat    uint64 = 0 //TODO: this should probably be a time.Duration
		leader           string
		isElectionNow           = false // this might not even really be needed
		term             uint32 = 0
		numPeers         int    = 0
		lastTermVotedFor uint32 = 0 // not even needed either?
		lastLogTerm      uint32 = 0
		lastLogIndex     uint64 = 0
	)

	var _ = lastTermVotedFor

	for {
		select {
		case message := <-rx.raftChannels.heartbeat:
			createdTime := message.GetCreatedTime()
			latency := uint64(time.Now().UnixNano()) - createdTime
			log.Println("got heartbeat(", createdTime, "), latency =", latency, "ns")
			fromName := message.GetName()
			if fromName == leader {
				lastHeartbeat = createdTime
			}
		case now := <-rx.raftChannels.electionTimeout:
			var _ = isElectionNow
			if lastHeartbeat+uint64(rx.heartbeatInterval) < now {
				log.Println("need to elect")
				isElectionNow = true
				term++
				rx.BroadcastHoldElection(term)
				// TODO: perhaps should use separate variable instead of abusing lastHeartbeat time
				lastHeartbeat = uint64(time.Now().UnixNano())
				// TODO: vote for self
			}
		case electionRequest := <-rx.raftChannels.holdElection:
			log.Println("got electionRequest", electionRequest)
			if *(electionRequest.Term) > term {
				isElectionNow = true
				term = *electionRequest.Term
				requestLastLogTerm := electionRequest.GetLastLogTerm()
				// TODO: perhaps should use separate variable instead of abusing lastHeartbeat time
				lastHeartbeat = uint64(time.Now().UnixNano())
				// this algorithm prefers higher last log terms, then last log index
				if requestLastLogTerm >= lastLogTerm {
					// prefer electionRequest.leader
					// vote for ^
				} else if requestLastLogTerm == lastLogTerm {
					if electionRequest.GetLastLogIndex() >= lastLogIndex {
						// prefer electionRequest.leader
					} else {
						// vote for current leader
					}
				} else {
					// vote for current leader
				}
			} // else do nothing
		case n := <-rx.raftChannels.numPeers:
			// tbh this variable is infrequently accessed. could be updated via atomics instead of message passing
			numPeers = n
			log.Println(numPeers, "peers")
			var _ = numPeers
		}
	}
}

type ConnChannels struct {
	newConn    chan *net.Conn
	removeConn chan *net.Conn
	broadcast  chan []byte
}

func NewConnChannels() *ConnChannels {
	return &ConnChannels{
		newConn:    make(chan *net.Conn),
		removeConn: make(chan *net.Conn),
		broadcast:  make(chan []byte, 32),
	}
}

// manages broadcasts and peer lists
func (rx *Server) connLoop() {
	peers := make(map[net.Addr]*net.Conn)
	for {
		select {
		case newConn := <-rx.connChannels.newConn:
			peers[(*newConn).RemoteAddr()] = newConn
			// update the raftstate
			rx.raftChannels.numPeers <- len(peers)
		case removeConn := <-rx.connChannels.removeConn:
			delete(peers, (*removeConn).RemoteAddr())
			rx.raftChannels.numPeers <- len(peers)
		case data := <-rx.connChannels.broadcast:
			for conn_id, conn := range peers {
				log.Println("sending", len(data), "bytes to", conn_id)
				// TODO: Conn.Write is ambiguous if partial writes are possible. should circle back
				_, err := (*conn).Write(data)
				if err != nil {
					rx.removePeer(conn)
				}
			}
		}
	}
}

type Server struct {
	Name                 string
	listener             net.Listener
	heartbeatInterval    time.Duration
	electionTimeout      time.Duration
	electionTimeoutDrift time.Duration
	raftChannels         *RaftChannels
	connChannels         *ConnChannels
}

func NewServer(port uint16, name string) (*Server, error) {
	address := fmt.Sprintf("0.0.0.0:%d", port)
	listener, err := net.Listen("tcp", address)
	log.Println("listening on", address)
	if err != nil {
		return nil, err
	} else {
		rx := Server{
			Name:                 name,
			listener:             listener,
			heartbeatInterval:    3 * time.Second,
			electionTimeout:      2 * time.Second,
			electionTimeoutDrift: 800 * time.Millisecond,
			raftChannels:         NewRaftChannels(),
			connChannels:         NewConnChannels(),
		}
		// coroutines for background stuff
		// accepts connections
		go rx.acceptLoop()

		// broadcasts heartbeats
		go rx.heartbeatLoop()

		// manages election/term related things
		go rx.electionTimeoutLoop()

		// handles fanned in channel messages related to raft
		go rx.raftLoop()

		// handles outbound broadcasts and peer connection state
		go rx.connLoop()

		return &rx, nil
	}
}

func (rx *Server) acceptLoop() error {
	for {
		conn, err := rx.listener.Accept()
		if err == nil {
			log.Println("peer connected", conn.RemoteAddr())
			go rx.recordPeer(&conn)
			go rx.handle(conn)
		} else {
			log.Println("ERROR accepting listener", err)
			return err
		}
	}
}

// sends heartbeats in a loop
func (rx *Server) heartbeatLoop() {
	for {
		rx.BroadcastHeartBeat()
		time.Sleep(rx.heartbeatInterval)
	}
}

func (rx *Server) electionTimeoutLoop() {
	for {
		// randomize election timeout to minimize colliding elections
		driftRange := 2 * rx.electionTimeoutDrift
		d := time.Duration(rand.Intn(int(driftRange)))
		sleepDuration := rx.electionTimeout - rx.electionTimeoutDrift + d
		//log.Println("sleep duration:", sleepDuration)
		time.Sleep(sleepDuration)
		rx.raftChannels.electionTimeout <- uint64(time.Now().UnixNano())
	}
}

// handles traffic on a connected (accepted) socket
func (rx *Server) handle(conn net.Conn) {
	// TODO: probably either: have a separate conn for larger messages
	//   OR: use a growable buffer
	buffer := make([]byte, 4096) // buffer size of 4KB
	for {
		// continuously read from conn
		bytesRead, err := conn.Read(buffer)
		if err != nil {
			rx.removePeer(&conn)
			log.Println("error reading", err)
			break
		}
		received := buffer[:bytesRead]
		log.Println("received", bytesRead, "bytes", received, "from", conn.RemoteAddr())
		var newMessage messages.PeerMessage
		proto.Unmarshal(received, &newMessage)
		rx.onReceive(&newMessage, conn)
	}
}

// reacts to a received protobuf PeerMessage
func (rx *Server) onReceive(message *messages.PeerMessage, conn net.Conn) {
	// log.Println("unmarshalled protobuf", *message)
	// due to the optional payloads, many message types can be composed into a single PeerMessage
	if message.GetHeartBeat() != nil {
		rx.raftChannels.heartbeat <- message.GetHeartBeat()
	}
	if message.GetPeerList() != nil {
		// do nothing
	}
	if message.GetHoldElection() != nil {
		rx.raftChannels.holdElection <- message.GetHoldElection()
	}
}

func (rx *Server) ConnectToPeer(addresses ...string) {
	// TODO: don't connect if it's already connected
	for _, address := range addresses {
		log.Println("connecting to peer", address)
		conn, err := net.Dial("tcp", address)
		if err != nil {
			log.Println("error connecting to peer", err)
		} else {
			go rx.handle(conn)
			go rx.recordPeer(&conn)
		}
	}
}

// records the existence of a peer, thread-safely
func (rx *Server) recordPeer(conn *net.Conn) {
	rx.connChannels.newConn <- conn
}

// closes the conn and removes a peer from the map
func (rx *Server) removePeer(conn *net.Conn) {
	rx.connChannels.removeConn <- conn
}

func (rx *Server) BroadcastHeartBeat() {
	rx.connChannels.broadcast <- rx.HeartBeatMessage()
}

func (rx *Server) BroadcastHoldElection(term uint32) {
	rx.connChannels.broadcast <- rx.HoldElectionMessage(term)
}

// TODO: ballots don't really need to be broadcast, just need to be sent to the candidate
// this isn't quite possible yet, without adding a name net.Conn map
func (rx *Server) BroadcastBallot(candidate string, term uint32) {
	rx.connChannels.broadcast <- rx.BallotMessage(&candidate, term)
}

func (rx *Server) HoldElectionMessage(term uint32) []byte {
	elect := &messages.HoldElection{
		Name: &rx.Name,
		Term: proto.Uint32(term),
	}
	data, _ := proto.Marshal(&messages.PeerMessage{
		HoldElection: elect,
	})
	return data
}

// serializers
func (rx *Server) HeartBeatMessage() []byte {
	heartBeat := &messages.HeartBeat{
		Name:        &rx.Name,
		CreatedTime: proto.Uint64(uint64(time.Now().UnixNano())),
	}
	data, _ := proto.Marshal(&messages.PeerMessage{
		HeartBeat: heartBeat,
	})
	return data
}

func (rx *Server) BallotMessage(candidate *string, term uint32) []byte {
	ballot := &messages.Ballot{
		Candidate: candidate,
		Term:      proto.Uint32(term),
	}
	data, _ := proto.Marshal(&messages.PeerMessage{
		Ballot: ballot,
	})
	return data
}

func startServer(port uint16, name string) *Server {
	server, err := NewServer(port, name)
	if err != nil {
		log.Fatal(err)
	}
	return server
}

func Start() {
	s1 := startServer(9001, "alice")
	s2 := startServer(9002, "bob")
	s3 := startServer(9003, "carol")

	s2.ConnectToPeer("0.0.0.0:9001", "0.0.0.0:9003")
	s3.ConnectToPeer("0.0.0.0:9001", "0.0.0.0:9002")

	var _ = s1

	<-make(chan bool)
}
