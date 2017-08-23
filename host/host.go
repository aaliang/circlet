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

type Server struct {
	listener net.Listener
	lock     sync.Mutex
	peers    map[net.Addr]*net.Conn // might not be necessary
}

func NewServer(port uint16) (*Server, error) {
	address := fmt.Sprintf("0.0.0.0:%d", port)
	listener, err := net.Listen("tcp", address)
	log.Println("listening on", address)
	if err != nil {
		return nil, err
	} else {
		rx := Server{
			listener: listener,
			lock:     sync.Mutex{},
			peers:    make(map[net.Addr]*net.Conn),
		}
		go rx.acceptLoop()
		return &rx, nil
	}
}

func (rx *Server) acceptLoop() error {
	for {
		conn, err := rx.listener.Accept()
		if err == nil {
			log.Println("peer connected", conn)
			go rx.recordPeer(&conn)
			go rx.handle(&conn)
		} else {
			log.Println("ERROR accepting listener", err)
			return err
		}
	}
}

func (rx *Server) handle(conn *net.Conn) {
	// TODO: probably either: have a separate conn for larger messages
	//   OR: use a growable buffer
	buffer := make([]byte, 4096) // buffer size of 4KB
	for {
		// continuously read from conn
		bytesRead, err := (*conn).Read(buffer)
		if err != nil {
			rx.removePeer(conn)
			break
		}
		received := buffer[:bytesRead]
		log.Println("received", bytesRead, "bytes", received)
		var newMessage messages.PeerMessage
		proto.Unmarshal(received, &newMessage)
		log.Println("unmarshalled protobuf", newMessage)
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
	log.Println("connecting to peer", address)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Println("error connecting to peer", err)
	} else {
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
	var _ = startServer(9001)
	s2 := startServer(9002)
	s2.ConnectToPeer("0.0.0.0:9001")
	s2.BroadcastHeartBeat()

	<-make(chan bool)
}
