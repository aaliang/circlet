package host

import (
	"fmt"
	"log"
	"net"
	"sync"
)

type Client struct{}

type Server struct {
	listener net.Listener
	lock     sync.Mutex
	peers    map[net.Addr]net.Conn // might not be necessary
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
			peers:    make(map[net.Addr]net.Conn),
		}
		go rx.acceptLoop()
		return &rx, nil
	}
}

func (rx *Server) acceptLoop() error {
	for {
		conn, err := rx.listener.Accept()
		log.Println("peer connected", conn)
		if err == nil {
			go rx.handle(&conn)
		} else {
			log.Println("ERROR accepting listener", err)
			return err
		}
	}
}

func (rx *Server) handle(conn *net.Conn) {

}

func (rx *Server) ConnectToPeer(address string) {

	log.Println("connecting to peer", address)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Println("error connecting to peer", err)
	} else {
		rx.lock.Lock()
		rx.peers[conn.RemoteAddr()] = conn
		rx.lock.Unlock()
	}
}

func startServer(port uint16) *Server {
	server, err := NewServer(port)
	if err != nil {
		log.Fatal(err)
	}
	return server
}

func Start() {
	var _ = startServer(9001)
	s2 := startServer(9002)
	s2.ConnectToPeer("0.0.0.0:9001")

	<-make(chan bool)
}
