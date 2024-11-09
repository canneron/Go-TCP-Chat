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

func newServer(conn net.Conn, nodes *[]model.Node) {
	defer conn.Close()

	message, err := bufio.NewReader(conn).ReadString('\n')
	message = strings.TrimSpace(message)
	if err != nil {
		fmt.Println("Error reading message:", err)
		return
	}

	// hostname, port, err := net.SplitHostPort(conn.LocalAddr().String())
	// if err != nil {
	// 	fmt.Println("Invalid port:", err)
	// 	return
	// }

	hostname, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	hostname = "[" + hostname + "]"

	newNode := model.Node{Hostname: hostname, Port: message}
	*nodes = append(*nodes, newNode)
	fmt.Println("New node connected:", newNode)

	address := hostname + ":" + message
	fmt.Println("add:", address)

	newConn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		os.Exit(1)
	}

	defer newConn.Close()
	fmt.Println("Sending connection list to", newNode.Address())

	networkConnections := "NODELIST!!"
	for _, node := range *nodes {
		if node == newNode {
			continue
		}
		networkConnections += fmt.Sprintf("%s:%s~~", node.Hostname, node.Port)
	}
	networkConnections += "\n"

	_, err = newConn.Write([]byte(networkConnections))
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	fmt.Println("Successfully added node to network")
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
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Error accepting connection:", err)
				continue
			}

			go newServer(conn, &connectedNodes)
		}
	}()

	wg.Wait()
}
