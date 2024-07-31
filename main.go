package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type Server struct {
	listenAddr string
	ln         net.Listener
	quitch     chan struct{}
	clients    map[*Client]struct{}
	lock       sync.Mutex
	history    []saveMsgs
}
type saveMsgs struct {
	name    string
	content string
}
type Client struct {
	conn net.Conn
	name string
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		clients:    make(map[*Client]struct{}),

		// msg:		make(chan []byte, 10),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()
	s.ln = ln
	go s.acceptLoop()
	<-s.quitch
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {

			fmt.Println("accept error:", err)
			continue
		}

		client := Client{conn: conn}
		s.addConn(&client)
		if len(s.clients) >= 4 {
			fmt.Println("max capacity has reached")
			return
		}
		welcomeMsg := "Welcome to TCP-Chat!\n[ENTER YOUR NAME]: "
		_, writeErr := conn.Write([]byte(welcomeMsg))

		if writeErr != nil {
			fmt.Println("Writting Error")
			conn.Close()
			continue
		}
		
		fmt.Println("Connection established with", conn.RemoteAddr())

		go s.readLoop(&client)

	}
}
func (s *Server) removeClient(client *Client) {

	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.clients, client)
}

func (s *Server) readLoop(client *Client) {
	defer client.conn.Close()
	msgs := []string{}
	buf := make([]byte, 2048)
	r := false

	for {
		n, err := client.conn.Read(buf)

		if err != nil {

			fmt.Println("read error :", err)

			s.broadcast(client, "Leaving Flag")
			s.removeClient(client)

			break
		}

		msg := buf[:n]

		msgs = append(msgs, string(msg))

		client.name = msgs[0]
		if r {
			s.broadcast(client, string(msg))
		} else {
			if len(s.clients) >= 2 {
				s.broadcast(client, "New XConectiom")
			}
			s.sendHistory(client)
		}
		_, writeErr := client.conn.Write([]byte(FormatTheMessage(client.name[:len(client.name)-1], string(msg[len(msg)-1]))))
		if writeErr != nil {
			fmt.Println("write error", err)
		}
		r = true
	}
}
func (s *Server) sendHistory(client *Client) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, msg := range s.history {
		historyMsg := FormatTheMessage(msg.name[:len(msg.name)-1], msg.content)
		
		_, err := client.conn.Write([]byte(historyMsg + msg.content))
		if err != nil {
			fmt.Println("Error sending history:", err)
			break
		}
	}
}
func (s *Server) broadcast(sender *Client, message string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if message != "New XConectiom" && message != "Leaving Flag" {
		s.history = append(s.history, saveMsgs{name: sender.name, content: message})
		// fmt.Println(s.history)
	}

	for client := range s.clients {
		if client != sender {
			if message == "New XConectiom" {
				// fmt.Println(client.name[:len(sender.name)])
				if len(client.name) > 0 {
					_, err := client.conn.Write([]byte("\n" + sender.name[:len(sender.name)-1] + " has joined our chat..." + "\n" + FormatTheMessage(client.name[:len(client.name)-1], "")))
					if err != nil {

						fmt.Println("error sending message:", err)

					}
					continue
				}
			} else if message == "Leaving Flag" {

				_, err := client.conn.Write([]byte("\n" + sender.name[:len(sender.name)-1] + " has left our chat..." + "\n" + FormatTheMessage(client.name[:len(client.name)-1], "")))
				if err != nil {

					fmt.Println("error sending message:", err)

				}

				continue
			}
			if len(client.name) > 0 {
				formattedMessage := "\n" + FormatTheMessage(sender.name[:len(sender.name)-1], message) + message + FormatTheMessage(client.name[:len(client.name)-1], "")

				_, err := client.conn.Write([]byte(formattedMessage))
				if err != nil {

					fmt.Println("error sending message:", err)

				}
			}
		}
	}

}

func (s *Server) addConn(client *Client) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.clients[client] = struct{}{}
}

func FormatTheMessage(userName, message string) string {
	currenttime := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("[%s][%s]: ", currenttime, userName)
}

func main() {
	Server := NewServer(":3000")
	log.Fatal(Server.Start())
}
