package model

import (
	"fmt"
	"strings"
)

type Node struct {
	Hostname string
	Port     string
	Nickname string
}

func (n Node) Address() string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(n.Hostname), strings.TrimSpace(n.Port))
}
