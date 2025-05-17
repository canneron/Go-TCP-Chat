package model

import "fmt"

type Channel struct {
	ConnectedNodes map[string]Node
	ChatHistory    []Message
	ChannelName    string
	ConnLimit      int
}

func NewChannel(name string) Channel {
	return Channel{
		ConnectedNodes: make(map[string]Node),
		ChatHistory:    []Message{},
		ChannelName:    name,
		ConnLimit:      100,
	}
}

func (c *Channel) ListMembers() {
	fmt.Printf("Connected %d/100:", len(c.ConnectedNodes))
	for _, node := range c.ConnectedNodes {
		fmt.Println(node.Nickname)
	}
}

func (c *Channel) PrintHistory() {
	for _, msg := range c.ChatHistory {
		fmt.Println(msg.PrintMessage())
	}
}
