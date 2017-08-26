package host

import (
	"circlet/messages"
	"fmt"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
	"time"
)

type RaftChannels struct {
	heartbeat       chan *messages.HeartBeat // receives heartbeats
	electionTimeout chan uint64              // periodic fixed interval to check for election timeouts. messages are a timestamp close to the current time
	holdElection    chan *messages.HoldElection
}

func (rx *Server) raftLoop() {
	//state
	var lastHeartbeat uint64 = 0
	var leader string
	var isElectionNow = false // this might not even really be needed
	var term uint32 = 0

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
			if !isElectionNow {
				if lastHeartbeat+uint64(rx.heartbeatInterval) < now {
					log.Println("need to elect")
					isElectionNow = true
					term++
					rx.BroadcastHoldElection(term)
					// TODO: vote for self
				} else {
					log.Println("pass")
				}
			}
		case electionRequest := <-rx.raftChannels.holdElection:
			if *(electionRequest.Term) > term {
				// need to vote
			} /*else {
				// do nothing
			}*/
		}
	}
}

type ConnChannels struct {
	newConn    chan *net.Conn
	removeConn chan *net.Conn
	broadcast  chan []byte
}

// manages broadcasts and peer lists
func (rx *Server) connLoop() {
	peers := make(map[net.Addr]*net.Conn)
	for {
		select {
		case newConn := <-rx.connChannels.newConn:
			peers[(*newConn).RemoteAddr()] = newConn
		case removeConn := <-rx.connChannels.removeConn:
			delete(peers, (*removeConn).RemoteAddr())
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
	Name              string
	listener          net.Listener
	heartbeatInterval time.Duration
	electionTimeout   time.Duration
	raftChannels      RaftChannels
	connChannels      ConnChannels
}

func NewServer(port uint16, name string) (*Server, error) {
	address := fmt.Sprintf("0.0.0.0:%d", port)
	listener, err := net.Listen("tcp", address)
	log.Println("listening on", address)
	if err != nil {
		return nil, err
	} else {
		rx := Server{
			Name:              name,
			listener:          listener,
			heartbeatInterval: 3 * time.Second,
			electionTimeout:   1 * time.Second,
			raftChannels: RaftChannels{
				heartbeat:       make(chan *messages.HeartBeat, 32),
				electionTimeout: make(chan uint64, 32),
				holdElection:    make(chan *messages.HoldElection, 32),
			},
			connChannels: ConnChannels{
				newConn:    make(chan *net.Conn),
				removeConn: make(chan *net.Conn),
				broadcast:  make(chan []byte, 32),
			},
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

// TODO: use randomized electionTimeout
func (rx *Server) electionTimeoutLoop() {
	for {
		time.Sleep(rx.electionTimeout)
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
	log.Println("unmarshalled protobuf", *message)
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

// records the existence of a peer, thread-safely
func (rx *Server) recordPeer(conn *net.Conn) {
	rx.connChannels.newConn <- conn
}

// closes the conn and removes a peer from the map
func (rx *Server) removePeer(conn *net.Conn) {
	rx.connChannels.removeConn <- conn
}

func (rx *Server) ConnectToPeer(address string) {
	// TODO: don't connect if it's already connected
	log.Println("connecting to peer", address)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Println("error connecting to peer", err)
	} else {
		go rx.handle(conn)
		rx.recordPeer(&conn)
	}
}

func (rx *Server) BroadcastHeartBeat() {
	rx.connChannels.broadcast <- rx.HeartBeatMessage()
}

func (rx *Server) BroadcastHoldElection(term uint32) {
	rx.connChannels.broadcast <- rx.HoldElectionMessage(term)
}

func startServer(port uint16, name string) *Server {
	server, err := NewServer(port, name)
	if err != nil {
		log.Fatal(err)
	}
	return server
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

func Start() {
	s1 := startServer(9001, "alice")
	s2 := startServer(9002, "bob")
	s2.ConnectToPeer("0.0.0.0:9001")

	var _ = s1

	<-make(chan bool)
}
