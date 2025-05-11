package model

import (
	"net"
)

type Channel struct {
	ConnectedNodes []Node
	Connections    []net.Conn
	ChatHistory    []Message
	ChannelName    string
	ConnLimit      int
}

func NewChannel(name string) Channel {
	return Channel{
		ConnectedNodes: []Node{},
		Connections:    []net.Conn{},
		ChatHistory:    []Message{},
		ChannelName:    name,
		ConnLimit:      100,
	}
}
