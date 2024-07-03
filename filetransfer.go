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
	port     int
}

func connectionServer(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Connection closed")
			return
		}

		fmt.Print("Received:", message)
		conn.Write([]byte("Message Received\n"))
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <hostname> <port>")
		return
	}

	hostname := os.Args[1]
	port := os.Args[2]

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		portLN := ":" + port
		listener, err := net.Listen("tcp", portLN)
		if err != nil {
			fmt.Println("Error starting TCP:", err)
			os.Exit(1)
		}

		defer listener.Close()

		fmt.Println("Listening on", hostname)
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Error accepting connection:", err)
				continue
			}

			go connectionServer(conn)
		}
	}()

	go func() {
		defer wg.Done()

		reader1 := bufio.NewReader(os.Stdin)
		fmt.Println("Enter port to connect to: ")
		portConnect, _ := reader1.ReadString('\n')
		portConnect = strings.TrimSpace(portConnect)
		serverInfo := fmt.Sprintf("%s:%s", hostname, portConnect)
		server, err := net.Dial("tcp", serverInfo)

		if err != nil {
			fmt.Println("Error connecting to server:", err)
			os.Exit(1)
		}
		//
		defer server.Close()

		fmt.Println("Connected to server")

		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Println("Enter message: ")
			text, _ := reader.ReadString('\n')
			fmt.Fprintf(server, text)

			message, _ := bufio.NewReader(server).ReadString('\n')
			fmt.Println("Response:", message)
		}
	}()

	wg.Wait()
}
