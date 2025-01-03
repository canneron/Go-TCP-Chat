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
	"time"
)

type Server struct {
	thisServer        model.Node
	knownNodes        []model.Node
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

		nodeInfo := fmt.Sprintf("NEW!!%s++%s\n", s.thisServer.Address(), s.thisServer.Nickname)

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
		newNodeInfo := strings.Split(node, "++")
		if newNodeInfo[0] == s.thisServer.Address() {
			s.thisServer.Nickname = newNodeInfo[1]
			continue
		} else {
			host, port, err := net.SplitHostPort(newNodeInfo[0])
			nickname := newNodeInfo[1]
			host = "[" + host + "]"
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			s.knownNodes = append(s.knownNodes, model.Node{Hostname: host, Port: port, Nickname: nickname})
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
	newNodeInfo := strings.Split(newNode, "++")
	host, port, err := net.SplitHostPort(newNodeInfo[0])
	nickname := newNodeInfo[1]
	host = "[" + host + "]"
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	node := model.Node{Hostname: host, Port: port, Nickname: nickname}
	s.knownNodes = append(s.knownNodes, node)
	s.connectToNode(node)
}

func constructMessage(body string) model.Message {
	msg := strings.Split(body, ";;")
	layout := "15:04:05"
	timestampStr := msg[0]
	parsedTime, _ := time.Parse(layout, timestampStr)
	return model.Message{Content: msg[2], Nickname: msg[1], Timestamp: parsedTime}
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

		fmt.Println("\nmsg", message)

		headerSplit := strings.Split(message, "!!")
		if len(headerSplit) < 2 {
			fmt.Println("Error processing message: ", headerSplit)
		}

		header := headerSplit[0]
		body := headerSplit[1]

		switch header {
		case "NODELIST":
			s.addNodes(body)
			var input sync.WaitGroup
			input.Add(1)

			go func() {
				defer input.Done()
				s.readInput()
			}()

			input.Wait()
		case "NEW":
			s.addNode(body)
		default:
			msg := constructMessage(body)
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
		text = "MSG!!" + msg.ConstructPacket()
		if text == "EXIT\n" {
			fmt.Println("Exit command received.")
			return
		}

		for _, node := range s.activeConnections {
			fmt.Fprintf(node, text)
		}

		conn, _ := net.Dial("tcp", s.thisServer.Address())
		fmt.Fprintf(conn, text)
	}
}

func connectToMirror(serverPort string, serverNickname string) {
	fmt.Println("Connecting to mirror 8080")
	mirror := "localhost:8080"

	mirrorConn, err := net.Dial("tcp", mirror)
	if err != nil {
		fmt.Println("Error connecting to mirror:", err)
		os.Exit(1)
	}

	defer mirrorConn.Close()

	port, _ := strconv.Atoi(serverPort)
	fmt.Fprintf(mirrorConn, "%d++%s\n", port, serverNickname)
	fmt.Println("Connected to network")
}

func (server *Server) start() {
	var wg sync.WaitGroup
	wg.Add(1)

	// ListenerBelfast
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

	connectToMirror(server.thisServer.Port, server.thisServer.Nickname)

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
	server := &Server{serverNode, []model.Node{}, []net.Conn{}, []model.Message{}}

	server.start()
}
