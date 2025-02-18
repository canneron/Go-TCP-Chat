package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go-p2p/model"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Server struct {
	thisServer        model.Node
	knownNodes        []model.Node
	knownMirrors      []model.Node
	activeConnections []net.Conn
	messageArchive    []model.Message
}

func (s *Server) networkBroadcast() {
	for _, node := range s.knownNodes {
		if node == s.thisServer {
			continue
		}
		conn := s.connectToNode(node)
		fmt.Println("Connected to", conn.RemoteAddr().String())

		nodeInfo := model.Message{Type: "NEW", Hostname: s.thisServer.Hostname, Port: s.thisServer.Port, Nickname: s.thisServer.Nickname, Timestamp: time.Now()}

		jsonData, err := json.Marshal(nodeInfo)
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			continue
		}

		_, err = conn.Write(jsonData)
		if err != nil {
			fmt.Println("Error sending message:", err)
			return
		}
		fmt.Println("Node information sent")
	}
}

func (s *Server) addNodes(nodelist []model.Node) {
	s.knownNodes = append(s.knownNodes, nodelist...)

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

func (s *Server) addNode(host string, port string, nickname string) {
	node := model.Node{Hostname: host, Port: port, Nickname: nickname}
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

		fmt.Println("\n", message)

		var msg model.Message
		if err := json.Unmarshal([]byte(message), &msg); err != nil {
			fmt.Println("Error decoding JSON:", err)
			return
		}
		header := msg.Type

		switch header {
		case "NEW":
			s.addNode(msg.Hostname, msg.Port, msg.Nickname)
		default:
			s.messageArchive = append(s.messageArchive, msg)
			fmt.Print(msg.PrintMessage())
		}
	}
}

func (s *Server) readInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter message: ")
		text, _ := reader.ReadString('\n')
		ts := time.Now()

		msg := model.Message{Content: text, Nickname: s.thisServer.Nickname, Timestamp: ts}

		if text == "EXIT\n" {
			fmt.Println("Exit command received.")
			return
		}

		jsonData, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			continue
		}

		for _, node := range s.activeConnections {
			node.Write(jsonData)
		}

		conn, _ := net.Dial("tcp", s.thisServer.Address())
		conn.Write(jsonData)
	}
}

func (s *Server) connectToMirror() {
	for _, mirror := range s.knownMirrors {
		fmt.Println("Connecting to ", mirror.Nickname)

		url := fmt.Sprintf("http://%s:%s/getNodes", mirror.Hostname, mirror.Port)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(s.thisServer.ToJson()))
		if err != nil {
			fmt.Println("Error sending request:", err)
			return
		}
		defer resp.Body.Close()

		var msg model.DiscoverMessage
		if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
			fmt.Println("Error decoding JSON:", err)
			return
		}

		s.addNodes(msg.NodeList)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		// Print the response
		fmt.Println("Response from server:", result)
		fmt.Println("Connected to network")

		var input sync.WaitGroup
		input.Add(1)

		go func() {
			defer input.Done()
			s.readInput()
		}()

		input.Wait()
	}
}

func (s *Server) loadMirrors() {
	file, err := os.Open("mirrorlist.txt")
	if err != nil {
		fmt.Println("Error opening mirrorlist.txt:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		mirrorInfo := strings.Split(line, "++")
		host, port, err := net.SplitHostPort(mirrorInfo[0])
		nickname := mirrorInfo[1]
		host = "[" + host + "]"
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		s.knownMirrors = append(s.knownMirrors, model.Node{Hostname: host, Port: port, Nickname: nickname})
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading mirrorlist.txt:", err)
	}
}

func (server *Server) start() {
	server.loadMirrors()
	var wg sync.WaitGroup
	wg.Add(1)

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

	server.connectToMirror()

	wg.Wait()
}

func chooseName() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter nickname: ")
	text, _ := reader.ReadString('\n')
	return text
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

	nickname := chooseName()

	serverNode := model.Node{Hostname: hostname, Port: port, Nickname: nickname}
	server := &Server{serverNode, []model.Node{}, []model.Node{}, []net.Conn{}, []model.Message{}}

	server.start()
}
