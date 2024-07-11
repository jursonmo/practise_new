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
}

type SSHMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
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

	address := request.Host + ":" + request.Port
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		ws.WriteJSON(SSHMessage{Type: "error", Payload: err.Error()})
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		ws.WriteJSON(SSHMessage{Type: "error", Payload: err.Error()})
		return
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		ws.WriteJSON(SSHMessage{Type: "error", Payload: err.Error()})
		return
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		ws.WriteJSON(SSHMessage{Type: "error", Payload: err.Error()})
		return
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		ws.WriteJSON(SSHMessage{Type: "error", Payload: err.Error()})
		return
	}

	if err := session.Shell(); err != nil {
		ws.WriteJSON(SSHMessage{Type: "error", Payload: err.Error()})
		return
	}

	go func() {
		buf := make([]byte, 2048)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				break
			}
			fmt.Printf("response:%s\n", string(buf[:n]))
			ws.WriteJSON(SSHMessage{Type: "output", Payload: string(buf[:n])})
		}
	}()

	for {
		var msg SSHMessage
		if err := ws.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Type == "input" {
			stdin.Write([]byte(msg.Payload))
		}
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

	var request SSHRequest
	if err := ws.ReadJSON(&request); err != nil {
		ws.WriteJSON(SSHMessage{Type: "error", Payload: "Invalid request"})
		return
	}

	handleSSH(ws, request)
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
