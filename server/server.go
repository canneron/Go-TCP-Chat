package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"go-p2p/certs"
	"go-p2p/enum/headerType"
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
	thisServer            model.Node
	knownNodes            map[string]*model.Node
	knownMirrors          []model.Node
	channels              map[string]*model.Channel
	privateMessageHistory map[string][]model.Message
}

var defaultChannelName = "lobby"
var defaultChannel = model.NewChannel(defaultChannelName)
var defaultChans = map[string]*model.Channel{
	defaultChannelName: &defaultChannel,
}

func (s *Server) connectToNode(node model.Node) net.Conn {
	fmt.Println("Connecting to node", node.Address())

	conn, err := tls.Dial("tcp", node.Address(), s.thisServer.ID.Config)
	if err != nil {
		fmt.Println("Error connecting to node:", err)
		os.Exit(1)
	}

	node.Connection = conn
	s.knownNodes[node.HashID()] = &node
	return conn
}

func (s *Server) addNode(host, port, nickname string) {
	fmt.Println("node: {} {} {}", host, port, nickname)
	node := model.Node{Hostname: host, Port: port, Nickname: nickname, Channel: defaultChannel}

	if _, exists := s.knownNodes[node.HashID()]; exists {
		return
	}

	s.connectToNode(node)
}

func (s *Server) handleChannel(channelList []model.Channel) {
	for _, chanName := range channelList {
		if _, exists := s.channels[chanName.ChannelName]; exists {
			continue
		} else {
			s.channels[chanName.ChannelName] = &chanName
		}
	}
}

func (s *Server) handleMessage(incomingMsg model.Message) {
	header := incomingMsg.Type

	switch header {
	// New Node
	case headerType.NewNode:
		var incomingChannel []model.Channel
		if err := json.Unmarshal([]byte(incomingMsg.Content), &incomingChannel); err != nil {
			fmt.Println("Error unmarshaling Content into Channel:", err)
			return
		}

		s.handleChannel(incomingChannel)
		s.addNode(incomingMsg.Hostname, incomingMsg.Port, incomingMsg.Nickname)
	// New Channel
	case headerType.NewChannel:
		s.updateNewChannel(incomingMsg.Content, incomingMsg.Nickname, incomingMsg.HashID)
	// Update Channel
	case headerType.UpdateChannel:
		s.updateChannelList(incomingMsg.Content, incomingMsg.Nickname, incomingMsg.HashID)
	// Channel Info
	case headerType.ChannelInfo:
		s.joinChannel(incomingMsg.Content)
	// Private Message
	case headerType.PrivateMessage:
		s.privateMessageHistory[incomingMsg.Nickname] = append(s.privateMessageHistory[incomingMsg.HashID], incomingMsg)
	// Exit Message
	case headerType.Exit:
		s.removeNode(incomingMsg.HashID)
	// Chat Message
	case headerType.ChatMessage:
		s.thisServer.Channel.ChatHistory = append(s.thisServer.Channel.ChatHistory, incomingMsg)
		fmt.Print(incomingMsg.PrintMessage())
	default:
		fmt.Println("Invalid message: {}", string(incomingMsg.ToJson()))
	}
}

func (s *Server) connectionServer(conn net.Conn) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)

	for {
		var incomingMsg model.Message
		err := decoder.Decode(&incomingMsg)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed by client:", conn.RemoteAddr().String())
				break
			} else {
				fmt.Println("Error reading message:", err)
				return
			}
		}

		s.handleMessage(incomingMsg)
	}
}

func (s *Server) networkBroadcast(nodeList []model.Node) {
	for _, node := range nodeList {
		if node.HashID() == s.thisServer.HashID() {
			continue
		}
		conn := s.connectToNode(node)
		node.Connection = conn
		fmt.Println("Connected to", conn.RemoteAddr().String())

		channelListJSON, err1 := json.Marshal(s.channels)

		if err1 != nil {
			fmt.Println("Error encoding channel JSON:", err1)
			continue
		}

		nodeInfo := model.Message{Type: headerType.NewNode, Hostname: s.thisServer.Hostname, Port: s.thisServer.Port, Nickname: s.thisServer.Nickname, Timestamp: time.Now(), Content: string(channelListJSON), HashID: s.thisServer.HashID()}

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

		s.networkBroadcast(msg.NodeList)

		fmt.Println("Response from server:", msg)
		fmt.Println("Connected to network")

		var input sync.WaitGroup
		input.Add(1)

		go func() {
			defer input.Done()
			s.sendMessageToChannel()
		}()

		input.Wait()
	}
}

func (s *Server) exit() {
	ts := time.Now()

	msg := model.Message{Type: headerType.Exit, Hostname: s.thisServer.Hostname, Port: s.thisServer.Port, Timestamp: ts, HashID: s.thisServer.HashID()}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	for _, node := range s.knownNodes {
		node.Connection.Write(jsonData)
	}

	fmt.Println("Server closing...")
	os.Exit(0)
}

func (s *Server) sendPrivateMessage(message, address string) {
	targetNode := s.knownNodes[address]

	ts := time.Now()

	msg := model.Message{Content: message, Nickname: s.thisServer.Nickname, Timestamp: ts, Type: headerType.PrivateMessage, HashID: s.thisServer.HashID()}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	targetNode.Connection.Write(jsonData)
}

// Channel Functions
func (s *Server) updateNewChannel(channel, nickname, hashId string) {
	if node, exists := s.knownNodes[hashId]; exists {
		node.Channel = model.NewChannel(channel)

		if _, exists := s.thisServer.Channel.ConnectedNodes[hashId]; exists {
			delete(s.thisServer.Channel.ConnectedNodes, node.Address())
			fmt.Printf("%s has left the channel.\n", nickname)
		}
	}

	if _, exists := s.channels[channel]; !exists {
		newChannel := model.NewChannel(channel)
		s.channels[channel] = &newChannel
	}
}

func (s *Server) updateChannelList(channel, nickname, hashId string) {
	if node, exists := s.knownNodes[hashId]; exists {
		if existingChan, exists := s.channels[node.Channel.ChannelName]; exists {
			if _, exists := existingChan.ConnectedNodes[hashId]; exists {
				delete(existingChan.ConnectedNodes, node.Address())
			}
		} else {
			fmt.Println("Channel not found:", node.Channel.ChannelName)
		}

		if newChannel, exists := s.channels[channel]; exists {
			newChannel.ConnectedNodes[node.Address()] = *node
		} else {
			fmt.Println("Channel not found:", channel)
		}

		if channel == s.thisServer.Channel.ChannelName {
			s.thisServer.Channel.ConnectedNodes[hashId] = *node
			fmt.Printf("%s has joined the channel.\n", nickname)

			channelInfo, _ := json.Marshal(s.thisServer.Channel)
			msg := model.Message{Type: headerType.ChannelInfo, Hostname: s.thisServer.Hostname, Port: s.thisServer.Port, Content: string(channelInfo), Nickname: s.thisServer.Nickname, Timestamp: time.Now(), HashID: s.thisServer.HashID()}
			jsonData, _ := json.Marshal(msg)
			node.Connection.Write(jsonData)
		} else if _, exists := s.thisServer.Channel.ConnectedNodes[hashId]; exists {
			delete(s.thisServer.Channel.ConnectedNodes, node.Address())
			fmt.Printf("%s has left the channel.\n", nickname)
		}
	}
}

func (s *Server) joinChannel(channel string) {
	var incomingChannel model.Channel
	if err := json.Unmarshal([]byte(channel), &incomingChannel); err != nil {
		fmt.Println("Error unmarshaling Content into Channel:", err)
		return
	}

	if channel != s.thisServer.Channel.ChannelName {
		s.thisServer.Channel = incomingChannel
		s.thisServer.Channel.OrderMessages()
		s.thisServer.Channel.PrintHistory()
		fmt.Printf("%s has joined the channel.\n", s.thisServer.Nickname)
	}
}

func (s *Server) sendMessageToChannel() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter message: ")
		text, _ := reader.ReadString('\n')
		ts := time.Now()

		msg := model.Message{Type: headerType.ChatMessage, Content: text, Nickname: s.thisServer.Nickname, Timestamp: ts, HashID: s.thisServer.HashID()}

		if text == "EXIT\n" {
			fmt.Println("Exit command received.")
			return
		}

		jsonData, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			continue
		}

		for _, node := range s.thisServer.Channel.ConnectedNodes {
			node.Connection.Write(jsonData)
		}

		conn, _ := tls.Dial("tcp", s.thisServer.Address(), s.thisServer.ID.Config)
		conn.Write(jsonData)
	}
}

func (server *Server) CreateChannel(name string) {
	server.thisServer.Channel = model.NewChannel(name)
	msg := model.Message{Type: headerType.NewChannel, Hostname: server.thisServer.Hostname, Port: server.thisServer.Port, Content: name, Nickname: server.thisServer.Nickname, Timestamp: time.Now(), HashID: server.thisServer.HashID()}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
	}

	for _, node := range server.knownNodes {
		node.Connection.Write(jsonData)
	}

	conn, _ := tls.Dial("tcp", server.thisServer.Address(), server.thisServer.ID.Config)
	conn.Write(jsonData)
}

func (server *Server) ChangeChannel(channel string) {
	server.thisServer.Channel = model.NewChannel(channel)
	msg := model.Message{Type: headerType.UpdateChannel, Hostname: server.thisServer.Hostname, Port: server.thisServer.Port, Content: channel, Nickname: server.thisServer.Nickname, Timestamp: time.Now(), HashID: server.thisServer.HashID()}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
	}

	for _, node := range server.knownNodes {
		node.Connection.Write(jsonData)
	}
}

func (server *Server) poll() {
	var wg sync.WaitGroup

	for hashId, node := range server.knownNodes {
		wg.Add(1)
		go server.pollNode(hashId, *node, &wg)
	}

	wg.Wait()
}

func (server *Server) pollNode(hashId string, node model.Node, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("Polling node:", node.Address())
	timeout := 7 * time.Second
	conn, err := net.DialTimeout("tcp", node.Address(), timeout)
	if err != nil {
		fmt.Println("Node timed out:", node.Address(), err)
		server.removeNode(hashId)
	} else {
		tlsConn := tls.Client(conn, server.thisServer.ID.Config)
		tlsConn.SetDeadline(time.Now().Add(timeout))
		err = tlsConn.Handshake()

		if err != nil {
			fmt.Println("TLS handshake failed:", node.Address(), err)
			server.removeNode(hashId)

			tlsConn.Close()
			conn.Close()
			return
		}
		tlsConn.Close()
		conn.Close()
	}
}

func (server *Server) removeNode(hashId string) {
	if node, exists := server.thisServer.Channel.ConnectedNodes[hashId]; exists {
		fmt.Println("Connection timed out:", node.Nickname)
		delete(server.thisServer.Channel.ConnectedNodes, hashId)
	} else {
		for _, channel := range server.channels {
			if _, exists := channel.ConnectedNodes[hashId]; exists {
				delete(channel.ConnectedNodes, hashId)
			}
		}
	}
	delete(server.knownNodes, hashId)
}

func (server *Server) startPolling(wg *sync.WaitGroup) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			server.poll()
		}
	}

	wg.Done()
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

		listener, err := tls.Listen("tcp", portLN, server.thisServer.ID.Config)
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

func generateIdentification(host, port string) *model.Identification {
	if _, err := os.Stat("../certs/key.pem"); errors.Is(err, os.ErrNotExist) {
		certs.GenerateCert(host, port)
	}

	keyStr, _ := os.ReadFile("../certs/key.pem")
	keyBlock, _ := pem.Decode(keyStr)
	pk, _ := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)

	certStr, _ := os.ReadFile("../certs/cert.pem")

	tlsCert, err := tls.X509KeyPair(certStr, keyStr)
	if err != nil {
		panic(err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	}

	return &model.Identification{PrivateKey: pk, Certificate: &tlsCert, Config: tlsConfig}
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
	identification := generateIdentification(hostname, port)

	serverNode := model.Node{Hostname: hostname, Port: port, Nickname: nickname, Channel: defaultChannel, ID: identification}

	server := &Server{serverNode, make(map[string]*model.Node), []model.Node{}, defaultChans, make(map[string][]model.Message)}

	var pollWG sync.WaitGroup
	pollWG.Add(1)
	go server.startPolling(&pollWG)
	server.start()

	pollWG.Wait()
}
