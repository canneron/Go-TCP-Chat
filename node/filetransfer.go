package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

type Node struct {
	hostname string
	port     string
}

func (n Node) Address() string {
	return fmt.Sprintf("%s:%s", n.hostname, n.port)
}

type Server struct {
	thisServer        Node
	knownNodes        []Node
	activeConnections []net.Conn
}

func splitNodeInfo(node string) (string, string, error) {
	parts := strings.Split(node, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format: expected 'hostname:port'")
	}
	return parts[0], parts[1], nil
}

func (s *Server) networkBroadcast() {
	for _, node := range s.knownNodes {
		conn := s.connectToNode(node)
		fmt.Println("Connected to", conn.LocalAddr().String())

		nodeInfo := "NEW!!"
		nodeInfo += s.thisServer.Address()

		_, err := conn.Write([]byte(nodeInfo))
		if err != nil {
			fmt.Println("Error sending message:", err)
			return
		}
		fmt.Println("Node information sent")
	}
}

func (s *Server) addNodes(nodeList string) {
	newNodes := strings.Split(nodeList, "~~")
	for _, node := range newNodes {
		host, port, err := splitNodeInfo(node)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		s.knownNodes = append(s.knownNodes, Node{host, port})
	}
	s.networkBroadcast()
}

func (s *Server) connectToNode(node Node) net.Conn {
	fmt.Println("Connecting to node", node.Address())
	conn, err := net.Dial("tcp", node.Address())
	if err != nil {
		fmt.Println("Error connecting to mirror:", err)
		os.Exit(1)
	}

	s.activeConnections = append(s.activeConnections, conn)
	return conn
}

func (s *Server) addNode(newNode string) {
	host, port, err := splitNodeInfo(newNode)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	node := Node{host, port}
	s.knownNodes = append(s.knownNodes, node)
	s.connectToNode(node)
}
func (s *Server) connectionServer(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Connection closed")
			return
		}

		headerSplit := strings.Split(message, "!!")
		if len(headerSplit) < 2 {
			fmt.Println("Error processing message: ", headerSplit)
		}
		header := headerSplit[0]
		body := headerSplit[1]

		switch header {
		case "NODELIST":
			s.addNodes(body)
		case "NEW":
			s.addNode(body)
		default:
			fmt.Print("Received:", message)
			conn.Write([]byte("Message Received\n"))
		}
	}
}

func connectToMirror() {
	fmt.Println("Connecting to mirror 8080")
	mirror := "localhost:8080"
	mirrorConn, err := net.Dial("tcp", mirror)
	if err != nil {
		fmt.Println("Error connecting to mirror:", err)
		os.Exit(1)
	}

	defer mirrorConn.Close()

	fmt.Println("Connected to network")
}
func (server *Server) start() {
	var wg sync.WaitGroup
	wg.Add(2)

	// Listener
	go func() {
		defer wg.Done()
		// Listener start up
		portLN := ":" + server.thisServer.port
		listener, err := net.Listen("tcp", portLN)
		if err != nil {
			fmt.Println("Error starting TCP:", err)
			os.Exit(1)
		}

		defer listener.Close()

		fmt.Println("Listening on ", server.thisServer.Address())
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Error accepting connection:", err)
				continue
			}

			go server.connectionServer(conn)
		}
	}()

	// Write to server
	// go func() {
	// 	defer wg.Done()

	// 	reader := bufio.NewReader(os.Stdin)
	// 	for {
	// 		fmt.Println("Enter message: ")
	// 		text, _ := reader.ReadString('\n')
	// 		fmt.Fprintf(server, text)

	// 		message, _ := bufio.NewReader(server).ReadString('\n')
	// 		fmt.Println("Response:", message)
	// 	}
	// }()

	connectToMirror()
	wg.Wait()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <hostname> <port>")
		return
	}

	hostname := os.Args[1]
	port := os.Args[2]
	serverNode := Node{hostname, port}
	server := &Server{serverNode, []Node{}, []net.Conn{}}

	server.start()
}
