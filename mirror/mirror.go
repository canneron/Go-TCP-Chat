package main

import (
	"fmt"
	"go-p2p/model"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func newServer(nodes *[]model.Node) model.DiscoverMessage {

	fmt.Println("Sending connection list: ", *nodes)

	msg := model.DiscoverMessage{NodeList: *nodes, Timestamp: time.Now()}

	return msg
}

func addServer(incomingNode model.Node, nodes *[]model.Node, nnMap *map[string]int) {
	nickname := incomingNode.Nickname

	if _, ok := (*nnMap)[incomingNode.Nickname]; ok {
		(*nnMap)[nickname]++
		nickname = fmt.Sprintf("%s(%d)", nickname, (*nnMap)[nickname])
	} else {
		(*nnMap)[nickname] = 0
	}

	newNode := model.Node{Hostname: incomingNode.Hostname, Port: incomingNode.Port, Nickname: nickname}
	*nodes = append(*nodes, newNode)
	fmt.Println("New node connected:", newNode)
}

func main() {
	fmt.Println("Starting mirror...")

	r := gin.Default()

	var connectedNodes []model.Node
	nicknameMap := make(map[string]int)

	r.POST("/getNodes", func(c *gin.Context) {
		var node model.Node
		if err := c.BindJSON(&node); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		addServer(node, &connectedNodes, &nicknameMap)
		c.JSON(http.StatusOK, newServer(&connectedNodes))
	})

	r.Run(":8080")

	fmt.Println("Stopping")
}
