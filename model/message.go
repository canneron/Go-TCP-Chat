package model

import (
	"fmt"
	"time"
)

type Message struct {
	Content   string
	Nickname  string
	Timestamp time.Time
}

func convertTime(ts time.Time) string {
	return ts.Format("15:04:05")
}

func (message Message) ConstructPacket() string {
	return fmt.Sprintf("%s;;%s;;%s", convertTime(message.Timestamp), message.Nickname, message.Content)
}

func (message Message) PrintMessage() string {
	return fmt.Sprintf("%s %s: %s", convertTime(message.Timestamp), message.Nickname, message.Content)
}
