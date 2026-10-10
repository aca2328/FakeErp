package main

import (
	"bytes"
	_ "embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var (
	//go:embed erp.html
	erpTmpl string
	//go:embed wsock.html
	wsockHTML []byte
	//go:embed favicon.ico
	favicon []byte
)

type site struct {
	Name  string
	Color string
}

// Per-site branding; selected with the SITE env var (default US).
var sites = map[string]site{
	"US": {"USA", "#0e9fc4"},
	"FR": {"FRANCE", "#c4810e"},
	"DE": {"GERMANY", "#750ec4"},
}

var upgrader = websocket.Upgrader{
	// Allowing connections from any origin
	CheckOrigin: func(r *http.Request) bool { return true },
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}
	defer conn.Close()

	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println("Failed to read message:", err)
			}
			return
		}
		if err := conn.WriteMessage(mt, message); err != nil {
			log.Println("Failed to write message:", err)
			return
		}
	}
}

func serveBytes(contentType string, body []byte) http.HandlerFunc {
	modTime := time.Now()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		http.ServeContent(w, r, "", modTime, bytes.NewReader(body))
	}
}

func main() {
	code := strings.ToUpper(os.Getenv("SITE"))
	s, ok := sites[code]
	if !ok {
		code, s = "US", sites["US"]
	}
	var buf bytes.Buffer
	if err := template.Must(template.New("erp").Parse(erpTmpl)).Execute(&buf, s); err != nil {
		log.Fatal(err)
	}
	log.Printf("Starting ERP site %s on :8080", code)

	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", serveBytes("text/html; charset=utf-8", buf.Bytes()))
	mux.HandleFunc("/wsock.html", serveBytes("text/html; charset=utf-8", wsockHTML))
	mux.HandleFunc("/favicon.ico", serveBytes("image/x-icon", favicon))
	mux.HandleFunc("/sock", echoHandler)

	mux.HandleFunc("/info", infoHandler(code))
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/health/fail", setHealth(false))
	mux.HandleFunc("/health/ok", setHealth(true))
	mux.HandleFunc("/slow", slowHandler)
	mux.HandleFunc("/error", errorHandler)
	mux.HandleFunc("/cpu", cpuHandler)
	mux.HandleFunc("/search", wafHandler)
	mux.HandleFunc("/login", wafHandler)

	srv := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
