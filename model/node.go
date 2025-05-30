package model

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

type Node struct {
	Hostname   string          `json:"hostname"`
	Port       string          `json:"port"`
	Nickname   string          `json:"nickname"`
	Connection net.Conn        `json:"-"`
	Channel    Channel         `json:"channel"`
	ID         *Identification `json:"-"`
}

func (n Node) Address() string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(n.Hostname), strings.TrimSpace(n.Port))
}

func (n Node) ToJson() []byte {
	jsonData, err := json.Marshal(n)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil
	}

	return jsonData
}
