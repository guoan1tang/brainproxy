package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWS(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)
	hub.Broadcast(`{"type":"test","data":"hello"}`)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != `{"type":"test","data":"hello"}` {
		t.Errorf("expected test message, got %s", string(msg))
	}
}

func TestHub_MultipleClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWS(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn1, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	conn2, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	defer conn1.Close()
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)
	hub.Broadcast(`{"type":"ping"}`)

	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))

	_, msg1, err1 := conn1.ReadMessage()
	_, msg2, err2 := conn2.ReadMessage()

	if err1 != nil || err2 != nil {
		t.Fatal("both clients should receive the message")
	}
	if string(msg1) != `{"type":"ping"}` || string(msg2) != `{"type":"ping"}` {
		t.Error("both clients should receive the same message")
	}
}
