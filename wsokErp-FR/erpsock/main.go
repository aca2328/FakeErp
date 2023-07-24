package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allowing connections from any origin
		return true
	},
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}
	defer conn.Close()

	for {
		// Read message from the client
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Failed to read message:", err)
			break
		}

		// Echo the received message back to the client
		err = conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println("Failed to write message:", err)
			break
		}
	}
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "erp.html")
		} else if r.URL.Path == "/wsock.html" {
			http.ServeFile(w, r, "wsock.html")
		} else if r.URL.Path == "/favicon.ico" {
			http.ServeFile(w, r, "favicon.ico")
		} else {
			http.NotFound(w, r)
		}
	})
	http.HandleFunc("/sock", echoHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

