package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Node struct {
	hostname string
	port     int
}

func newServer(conn net.Conn, nodes *[]Node) {
	defer conn.Close()

	connInfo := strings.Split(conn.RemoteAddr().String(), ":")
	if len(connInfo) != 2 {
		fmt.Println("Invalid address format")
		return
	}

	hostname := connInfo[0]
	port, err := strconv.Atoi(connInfo[1])
	if err != nil {
		fmt.Println("Invalid port:", err)
		return
	}

	*nodes = append(*nodes, Node{hostname, port})
	fmt.Println("New node connected:", *nodes)

	newConn, err := net.Dial("tcp", conn.LocalAddr().String())
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		os.Exit(1)
	}

	defer newConn.Close()
	fmt.Println("Sending connection list to", conn.LocalAddr().String())

	networkConnections := "NODELIST!!"
	for _, node := range *nodes {
		networkConnections += fmt.Sprintf("%s:%d~~", node.hostname, node.port)
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
		var connectedNodes []Node
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
