package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"
)

type client struct {
	ws   *websocket.Conn
	room string
}

var (
	rooms    = map[string][]*client{}
	roomsMtx = sync.Mutex{}
)

func wsHandler(ws *websocket.Conn) {
	defer ws.Close()
	var c client
	c.ws = ws

	for {
		var msg string
		if err := websocket.Message.Receive(ws, &msg); err != nil {
			roomsMtx.Lock()
			arr := rooms[c.room]
			for i, cl := range arr {
				if cl.ws == ws {
					rooms[c.room] = append(arr[:i], arr[i+1:]...)
					break
				}
			}
			roomsMtx.Unlock()
			return
		}

		if c.room == "" && (len(msg) > 8 && msg[:8] == `{"type":"`) {
			if contains(msg, `"type":"join"`) {
				room := extractRoom(msg)
				if room != "" {
					c.room = room
					roomsMtx.Lock()
					rooms[room] = append(rooms[room], &c)
					roomsMtx.Unlock()
					fmt.Printf("joined room %s\n", room)
					continue
				}
			}
		}

		if c.room != "" {
			roomsMtx.Lock()
			for _, cl := range rooms[c.room] {
				if cl.ws != ws {
					websocket.Message.Send(cl.ws, msg)
				}
			}
			roomsMtx.Unlock()
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func extractRoom(s string) string {
	start := indexOf(s, `"room":"`)
	if start == -1 {
		return ""
	}
	start += len(`"room":"`)
	end := indexOf(s[start:], `"`)
	if end == -1 {
		return ""
	}
	return s[start : start+end]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.Handle("/ws", websocket.Handler(wsHandler))
	fmt.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
