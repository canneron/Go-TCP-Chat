package model

type Channel struct {
	ConnectedNodes []Node
	ChatHistory    []Message
	ChannelName    string
	ConnLimit      int
}

func NewChannel(name string) Channel {
	return Channel{
		ConnectedNodes: []Node{},
		ChatHistory:    []Message{},
		ChannelName:    name,
		ConnLimit:      100,
	}
}
