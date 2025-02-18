package model

import (
	"time"
)

type DiscoverMessage struct {
	NodeList  []Node    `json:"nodelist"`
	Timestamp time.Time `json:"timestamp"`
}
