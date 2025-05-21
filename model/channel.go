package model

import (
	"fmt"
	"sort"
)

type Channel struct {
	ConnectedNodes map[string]Node `json:"connectedNodes"`
	ChatHistory    []Message       `json:"chatHistory"`
	ChannelName    string          `json:"channelName"`
	ConnLimit      int             `json:"-"`
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

func (c *Channel) OrderMessages() {
	sort.Slice(c.ChatHistory, func(i, j int) bool {
		return c.ChatHistory[i].Timestamp.After(c.ChatHistory[j].Timestamp)
	})
}
