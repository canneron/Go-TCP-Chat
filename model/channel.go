package model

import (
	"net"
)

type Channel struct {
	connectedNodes []Node
	connections    []net.Conn
	chatHistory    []Message
	channelName    string
	connLimit      int
}

func NewChannel(name string) Channel {
	return Channel{
		connectedNodes: []Node{},
		connections:    []net.Conn{},
		chatHistory:    []Message{},
		channelName:    name,
		connLimit:      100,
	}
}
