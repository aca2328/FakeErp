package main

// In-memory sessions, local to each backend, to demo cookie persistence: if
// the load balancer keeps the client on the same server, the counter keeps
// growing; if not, the session is "lost" and a new one is created.

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const sessionCookie = "ERPSESSION"

type session struct {
	User     string
	Created  time.Time
	Requests int
}

var (
	sessionsMu sync.Mutex
	sessions   = map[string]*session{}
)

func newSession(w http.ResponseWriter, user string) *session {
	b := make([]byte, 16)
	rand.Read(b)
	id := hex.EncodeToString(b)
	s := &session{User: user, Created: time.Now()}

	sessionsMu.Lock()
	if len(sessions) >= 10000 { // keep memory bounded under load tests
		clear(sessions)
	}
	sessions[id] = s
	sessionsMu.Unlock()

	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: id, Path: "/", HttpOnly: true})
	return s
}

func sessionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	status := "existing session"
	var s *session
	if c, err := r.Cookie(sessionCookie); err == nil {
		sessionsMu.Lock()
		s = sessions[c.Value]
		sessionsMu.Unlock()
		if s == nil {
			status = "SESSION LOST: cookie not known on this server (no persistence?) - new session created"
		}
	} else {
		status = "new session created"
	}
	if s == nil {
		s = newSession(w, "anonymous")
	}

	sessionsMu.Lock()
	s.Requests++
	n := s.Requests
	sessionsMu.Unlock()

	user := s.User
	if user == "" {
		user = "anonymous"
	}
	fmt.Fprintf(w, "%s\n\nserver:   %s (site %s)\nuser:     %s\ncreated:  %s\nrequests: %d on this server\n\nReload to check persistence. Log in as demo/demo at /search.\n",
		status, hostname, siteCode, user, s.Created.Format(time.RFC3339), n)
}
