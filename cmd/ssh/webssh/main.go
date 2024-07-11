package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

type SSHRequest struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Command  string `json:"command"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleSSH(ws *websocket.Conn, request SSHRequest) {
	config := &ssh.ClientConfig{
		User: request.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(request.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	address := fmt.Sprintf("%s:%s", request.Host, request.Port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		ws.WriteJSON(map[string]string{"error": err.Error()})
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		ws.WriteJSON(map[string]string{"error": err.Error()})
		return
	}
	defer session.Close()

	output, err := session.CombinedOutput(request.Command)
	if err != nil {
		ws.WriteJSON(map[string]string{"error": err.Error()})
		return
	}

	ws.WriteJSON(map[string]string{"output": string(output)})
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

	for {
		var request SSHRequest
		err := ws.ReadJSON(&request)
		if err != nil {
			log.Println(err)
			break
		}

		handleSSH(ws, request)
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
