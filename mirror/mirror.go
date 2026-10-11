package main

import (
	"encoding/pem"
	"fmt"
	"go-p2p/certs"
	"go-p2p/model"
	"net/http"
	"os"
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

	caCert, caKey, err := certs.LoadOrGenerateCA()
	if err != nil {
		fmt.Println("Error loading CA:", err)
		os.Exit(1)
	}
	fmt.Println("Loaded CA:", caCert.Subject.CommonName)
	fmt.Println("CA fingerprint (add to mirrorlist.txt):", certs.Fingerprint(caCert))
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw})

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

	r.GET("/caCert", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/x-pem-file", caCertPEM)
	})

	r.POST("/signCert", func(c *gin.Context) {
		csrPEM, err := c.GetRawData()
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		certPEM, err := certs.SignCSR(csrPEM, caCert, caKey)
		if err != nil {
			fmt.Println("Rejected CSR:", err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		fmt.Println("Signed cert for", c.ClientIP())
		c.Data(http.StatusOK, "application/x-pem-file", certPEM)
	})

	r.Run(":8080")

	fmt.Println("Stopping")
}
