package main

import (
	"bufio"
	"fmt"
	"go-p2p/model"
	"net"
	"os"
	"strings"
	"sync"
)

func newServer(conn net.Conn, nodes *[]model.Node, nnMap *map[string]int) {
	defer conn.Close()

	message, err := bufio.NewReader(conn).ReadString('\n')
	message = strings.TrimSpace(message)
	if err != nil {
		fmt.Println("Error reading message:", err)
		return
	}

	nodeInfo := strings.Split(message, "++")
	port := nodeInfo[0]
	nickname := nodeInfo[1]

	if _, ok := (*nnMap)[nickname]; ok {
		(*nnMap)[nickname]++
		nickname = fmt.Sprintf("%s(%d)", nickname, (*nnMap)[nickname])
	} else {
		(*nnMap)[nickname] = 0
	}

	hostname, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	hostname = "[" + hostname + "]"

	newNode := model.Node{Hostname: hostname, Port: port, Nickname: nickname}
	*nodes = append(*nodes, newNode)
	fmt.Println("New node connected:", newNode)

	address := hostname + ":" + port

	newConn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		os.Exit(1)
	}

	defer newConn.Close()
	fmt.Println("Sending connection list to", newNode.Address())

	networkConnections := "NODELIST!!"
	for _, node := range *nodes {
		networkConnections += fmt.Sprintf("%s:%s++%s~~", node.Hostname, node.Port, node.Nickname)
	}
	networkConnections += "\n"

	_, err = newConn.Write([]byte(networkConnections))
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	fmt.Println("Successfully added node to network\n")
}

func main() {
	fmt.Println("Starting mirror...")

	hostname := "localhost"
	port := "8080"

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		portLN := ":" + port
		listener, err := net.Listen("tcp", portLN)
		if err != nil {
			fmt.Println("Error starting TCP:", err)
			os.Exit(1)
		}

		defer listener.Close()

		fmt.Println("Listening on ", hostname, ":", port)
		var connectedNodes []model.Node
		nicknameMap := make(map[string]int)
		for {
			conn, err := listener.Accept()
			fmt.Println("***** Incoming Node! *****")
			if err != nil {
				fmt.Println("Error accepting connection:", err)
				continue
			}

			go newServer(conn, &connectedNodes, &nicknameMap)
		}
	}()

	wg.Wait()
}
