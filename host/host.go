package host

import (
	"circlet/messages"
	"fmt"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
	"sync"
	"time"
)

type Channels struct {
	heartbeat chan struct {
		payload *messages.HeartBeat
		conn    net.Conn
	}
}

func (rx *Server) channelLoop() {
	for {
		select {
		case message := <-rx.channels.heartbeat:
			//TODO: maybe delegate to a channel
			createdTime := message.payload.GetCreatedTime()
			latency := uint64(time.Now().UnixNano()) - createdTime
			log.Println("got heartbeat(", createdTime, "), latency =", latency, "ns")
		}
	}
}

type Server struct {
	listener          net.Listener
	lock              sync.Mutex
	peers             map[net.Addr]*net.Conn // might not be necessary
	heartbeatInterval time.Duration
	term              uint32
	electionTimeout   time.Duration
	channels          Channels
}

func NewServer(port uint16) (*Server, error) {
	address := fmt.Sprintf("0.0.0.0:%d", port)
	listener, err := net.Listen("tcp", address)
	log.Println("listening on", address)
	if err != nil {
		return nil, err
	} else {
		rx := Server{
			listener:          listener,
			lock:              sync.Mutex{},
			peers:             make(map[net.Addr]*net.Conn),
			heartbeatInterval: 3 * time.Second,
			term:              0,
			electionTimeout:   1 * time.Second,
			channels: Channels{
				heartbeat: make(chan struct {
					payload *messages.HeartBeat
					conn    net.Conn
				}),
			},
		}
		// coroutines for background stuff
		// accepts connections
		go rx.acceptLoop()

		// broadcasts heartbeats
		go rx.heartbeatLoop()

		// manages election/term related things
		go rx.termLoop()

		// handles fanned in channel messages
		go rx.channelLoop()

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

func (rx *Server) termLoop() {
	// check every few seconds when the last heartbeat received from the leader was
	//var isLeader bool = false
	for {
		time.Sleep(rx.electionTimeout)

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
		rx.channels.heartbeat <- struct {
			payload *messages.HeartBeat
			conn    net.Conn
		}{message.GetHeartBeat(), conn}
	}
	if message.GetPeerList() != nil {
	}
}

// records the existence of a peer, thread-safely
func (rx *Server) recordPeer(conn *net.Conn) {
	rx.lock.Lock()
	rx.peers[(*conn).RemoteAddr()] = conn
	rx.lock.Unlock()
}

// closes the conn and removes a peer from the map
func (rx *Server) removePeer(conn *net.Conn) {
	(*conn).Close()
	rx.lock.Lock()
	delete(rx.peers, (*conn).RemoteAddr())
	rx.lock.Unlock()
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

func (rx *Server) Broadcast(data []byte) {
	for conn_id, conn := range rx.peers {
		log.Println("sending", len(data), "bytes to", conn_id)
		// TODO: Conn.Write is ambiguous if partial writes are possible. should circle back
		_, err := (*conn).Write(data)
		if err != nil {
			rx.removePeer(conn)
		}
	}
}

func (rx *Server) BroadcastHeartBeat() {
	rx.Broadcast(HeartBeatMessage())
}

func startServer(port uint16) *Server {
	server, err := NewServer(port)
	if err != nil {
		log.Fatal(err)
	}
	return server
}

// serializers
func HeartBeatMessage() []byte {
	heartBeat := &messages.HeartBeat{
		CreatedTime: proto.Uint64(uint64(time.Now().UnixNano())),
	}
	data, _ := proto.Marshal(&messages.PeerMessage{
		HeartBeat: heartBeat,
	})
	return data
}

func Start() {
	s1 := startServer(9001)
	s2 := startServer(9002)
	s2.ConnectToPeer("0.0.0.0:9001")

	var _ = s1

	<-make(chan bool)
}
