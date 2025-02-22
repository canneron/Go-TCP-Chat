package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Message struct {
	Type      string    `json:"type"`
	Hostname  string    `json:"hostname"`
	Port      string    `json:"port"`
	Content   string    `json:"content"`
	Nickname  string    `json:"nickname"`
	Timestamp time.Time `json:"timestamp"`
}

func convertTime(ts time.Time) string {
	return ts.Format("15:04:05")
}

func (message Message) PrintMessage() string {
	cleanNN := strings.ReplaceAll(message.Nickname, "\n", "")
	return fmt.Sprintf("%s %s: %s", convertTime(message.Timestamp), cleanNN, message.Content)
}

func (message Message) ConstructPacket() string {
	return fmt.Sprintf("%s;;%s;;%s", convertTime(message.Timestamp), message.Nickname, message.Content)
}

func (message Message) toJson() []byte {
	jsonData, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil
	}

	return jsonData
}
