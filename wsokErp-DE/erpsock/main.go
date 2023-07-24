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
        switch r.URL.Path {
        case "/":
            http.ServeFile(w, r, "erp.html")
        case "/wsock.html":
            http.ServeFile(w, r, "wsock.html")
        case "/favicon.ico":
            http.ServeFile(w, r, "favicon.ico")
        default:
            http.NotFound(w, r)
        }
    })
    http.HandleFunc("/sock", echoHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
