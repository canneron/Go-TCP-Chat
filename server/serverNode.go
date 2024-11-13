package main

import (
	"bufio"
	"fmt"
	"go-p2p/model"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Server struct {
	thisServer        model.Node
	knownNodes        []model.Node
	activeConnections []net.Conn
}

func (s *Server) networkBroadcast() {
	for _, node := range s.knownNodes {
		if node == s.thisServer {
			continue
		}
		conn := s.connectToNode(node)
		fmt.Println("Connected to", conn.RemoteAddr().String())

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
	nodeList = strings.TrimSpace(nodeList)
	nodeList = strings.TrimSuffix(nodeList, "~~")
	newNodes := strings.Split(nodeList, "~~")
	fmt.Println("New Nodes:", nodeList)
	for _, node := range newNodes {
		fmt.Println(node + ":" + s.thisServer.Address())
		if node == s.thisServer.Address() {
			continue
		} else {
			host, port, err := net.SplitHostPort(node)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			s.knownNodes = append(s.knownNodes, model.Node{Hostname: host, Port: port})
		}
	}
	s.networkBroadcast()
}

func (s *Server) connectToNode(node model.Node) net.Conn {
	fmt.Println("Connecting to node", node.Address())
	conn, err := net.Dial("tcp", node.Address())
	if err != nil {
		fmt.Println("Error connecting to node:", err)
		os.Exit(1)
	}

	s.activeConnections = append(s.activeConnections, conn)
	return conn
}

func (s *Server) addNode(newNode string) {
	host, port, err := net.SplitHostPort(newNode)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	node := model.Node{Hostname: host, Port: port}
	s.knownNodes = append(s.knownNodes, node)
	s.connectToNode(node)
}

func (s *Server) connectionServer(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed by client:", conn.RemoteAddr().String())
				break
			} else {
				fmt.Println("Error reading message:", err)
				return
			}
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

func connectToMirror(serverPort string) {
	fmt.Println("Connecting to mirror 8080")
	mirror := "localhost:8080"

	mirrorConn, err := net.Dial("tcp", mirror)
	if err != nil {
		fmt.Println("Error connecting to mirror:", err)
		os.Exit(1)
	}

	defer mirrorConn.Close()

	port, _ := strconv.Atoi(serverPort)
	fmt.Fprintf(mirrorConn, "%d\n", port)
	fmt.Println("Connected to network")
}

func (server *Server) start() {
	var wg sync.WaitGroup
	wg.Add(2)

	// Listener
	go func() {
		defer wg.Done()
		// Listener start up
		portLN := ":" + server.thisServer.Port
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
	go func() {
		defer wg.Done()

		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Println("Enter message: ")
			text, _ := reader.ReadString('\n')
			for _, node := range server.knownNodes {
				server, err := net.Dial("tcp", node.Address())
				if err != nil {
					fmt.Println("Error connecting to server:", node.Address())
					continue
				}
				fmt.Fprintf(server, text)
			}
		}
	}()

	connectToMirror(server.thisServer.Port)
	wg.Wait()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <hostname> <port>")
		return
	}

	hostname := os.Args[1]
	port := os.Args[2]
	if hostname == "localhost" {
		hostname = "[::1]"
	}
	serverNode := model.Node{Hostname: hostname, Port: port}
	server := &Server{serverNode, []model.Node{}, []net.Conn{}}

	server.start()
}
