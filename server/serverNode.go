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

	conn, err := net.Dial("tcp", node.Address())
	if err != nil {
		fmt.Println("Error connecting to node:", err)
		os.Exit(1)
	}

	node.Connection = conn
	s.knownNodes[node.Address()] = &node
	return conn
}

func (s *Server) addNode(host string, port string, nickname string) {
	fmt.Println("node: {} {} {}", host, port, nickname)
	node := model.Node{Hostname: host, Port: port, Nickname: nickname, Channel: defaultChannel}

	if _, exists := s.knownNodes[node.Address()]; exists {
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
	case "NEW":
		var incomingChannel []model.Channel
		if err := json.Unmarshal([]byte(incomingMsg.Content), &incomingChannel); err != nil {
			fmt.Println("Error unmarshaling Content into Channel:", err)
			return
		}

		s.handleChannel(incomingChannel)
		s.addNode(incomingMsg.Hostname, incomingMsg.Port, incomingMsg.Nickname)
	case "NEW CHANNEL":
		s.updateNewChannel(incomingMsg.Content, incomingMsg.Hostname, incomingMsg.Port, incomingMsg.Nickname)
	case "UPDATE CHANNEL":
		s.updateChannelList(incomingMsg.Content, incomingMsg.Hostname, incomingMsg.Port, incomingMsg.Nickname)
	case "CHANNEL INFO":
		s.joinChannel(incomingMsg.Content)
	case "PM":
		s.privateMessageHistory[incomingMsg.Nickname] = append(s.privateMessageHistory[incomingMsg.Nickname], incomingMsg)
	case "EXIT":
		address := fmt.Sprintf("%s:%s", incomingMsg.Hostname, incomingMsg.Port)
		s.removeNode(address)
	default:
		s.thisServer.Channel.ChatHistory = append(s.thisServer.Channel.ChatHistory, incomingMsg)
		fmt.Print(incomingMsg.PrintMessage())
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
		if node.Hostname == s.thisServer.Hostname && node.Port == s.thisServer.Port {
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

		nodeInfo := model.Message{Type: "NEW", Hostname: s.thisServer.Hostname, Port: s.thisServer.Port, Nickname: s.thisServer.Nickname, Timestamp: time.Now(), Content: string(channelListJSON)}

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

	msg := model.Message{Type: "EXIT", Hostname: s.thisServer.Hostname, Port: s.thisServer.Port, Timestamp: ts}

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

func (s *Server) sendPrivateMessage(message string, address string) {
	targetNode := s.knownNodes[address]

	ts := time.Now()

	msg := model.Message{Content: message, Nickname: s.thisServer.Nickname, Timestamp: ts, Type: "PM"}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	targetNode.Connection.Write(jsonData)
}

// Channel Functions
func (s *Server) updateNewChannel(channel string, hostname string, nickname string, port string) {
	address := hostname + ":" + port
	if node, exists := s.knownNodes[address]; exists {
		node.Channel = model.NewChannel(channel)

		if _, exists := s.thisServer.Channel.ConnectedNodes[address]; exists {
			delete(s.thisServer.Channel.ConnectedNodes, node.Address())
			fmt.Printf("%s has left the channel.\n", nickname)
		}
	}

	if _, exists := s.channels[channel]; !exists {
		newChannel := model.NewChannel(channel)
		s.channels[channel] = &newChannel
	}
}

func (s *Server) updateChannelList(channel string, hostname string, nickname string, port string) {
	address := hostname + ":" + port
	if node, exists := s.knownNodes[address]; exists {
		if existingChan, exists := s.channels[node.Channel.ChannelName]; exists {
			if _, exists := existingChan.ConnectedNodes[address]; exists {
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
			s.thisServer.Channel.ConnectedNodes[address] = *node
			fmt.Printf("%s has joined the channel.\n", nickname)

			channelInfo, _ := json.Marshal(s.thisServer.Channel)
			msg := model.Message{Type: "CHANNEL INFO", Hostname: s.thisServer.Hostname, Port: s.thisServer.Port, Content: string(channelInfo), Nickname: s.thisServer.Nickname, Timestamp: time.Now()}
			jsonData, _ := json.Marshal(msg)
			node.Connection.Write(jsonData)
		} else if _, exists := s.thisServer.Channel.ConnectedNodes[address]; exists {
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

		for _, node := range s.thisServer.Channel.ConnectedNodes {
			node.Connection.Write(jsonData)
		}

		conn, _ := net.Dial("tcp", s.thisServer.Address())
		conn.Write(jsonData)
	}
}

func (server *Server) CreateChannel(name string) {
	server.thisServer.Channel = model.NewChannel(name)
	msg := model.Message{Type: "NEW CHANNEL", Hostname: server.thisServer.Hostname, Port: server.thisServer.Port, Content: name, Nickname: server.thisServer.Nickname, Timestamp: time.Now()}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
	}

	for _, node := range server.knownNodes {
		node.Connection.Write(jsonData)
	}

	conn, _ := net.Dial("tcp", server.thisServer.Address())
	conn.Write(jsonData)
}

func (server *Server) ChangeChannel(channel string) {
	server.thisServer.Channel = model.NewChannel(channel)
	msg := model.Message{Type: "UPDATE CHANNEL", Hostname: server.thisServer.Hostname, Port: server.thisServer.Port, Content: channel, Nickname: server.thisServer.Nickname, Timestamp: time.Now()}

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

	for address, node := range server.knownNodes {
		wg.Add(1)
		go server.pollNode(address, *node, &wg)
	}

	wg.Wait()
}

func (server *Server) pollNode(address string, node model.Node, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("Polling node:", address)
	timeout := 7 * time.Second
	conn, err := net.DialTimeout("tcp", node.Address(), timeout)
	if err != nil {
		fmt.Println("Node timed out:", address, err)
		server.removeNode(address)
	} else {
		conn.Close()
	}
}

func (server *Server) removeNode(address string) {
	if node, exists := server.thisServer.Channel.ConnectedNodes[address]; exists {
		fmt.Println("Connection timed out:", node.Nickname)
		delete(server.thisServer.Channel.ConnectedNodes, address)
	} else {
		for _, channel := range server.channels {
			if _, exists := channel.ConnectedNodes[address]; exists {
				delete(channel.ConnectedNodes, address)
			}
		}
	}
	delete(server.knownNodes, address)
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

	serverNode := model.Node{Hostname: hostname, Port: port, Nickname: nickname, Channel: defaultChannel}

	server := &Server{serverNode, make(map[string]*model.Node), []model.Node{}, defaultChans, make(map[string][]model.Message)}

	var pollWG sync.WaitGroup
	pollWG.Add(1)
	go server.startPolling(&pollWG)
	server.start()

	pollWG.Wait()
}
