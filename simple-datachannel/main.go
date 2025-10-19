package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/pion/webrtc/v4"
)

var (
	mu sync.Mutex
	pc *webrtc.PeerConnection
	dc *webrtc.DataChannel
)

func main() {
	// /offer endpoint
	http.HandleFunc("/offer", func(w http.ResponseWriter, r *http.Request) {
		var offer webrtc.SessionDescription
		if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// PeerConnection
		var err error
		pc, err = webrtc.NewPeerConnection(webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{URLs: []string{"stun:stun.l.google.com:19302"}},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// ICE candidate
		pc.OnICECandidate(func(c *webrtc.ICECandidate) {
			if c != nil {
				fmt.Printf("🌐 New ICE candidate: %s\n", c.Address)
			}
		})

		// DataChannel
		pc.OnDataChannel(func(d *webrtc.DataChannel) {
			dc = d
			d.OnOpen(func() {
				fmt.Println("✅ DataChannel opened (Server)")
				d.SendText("Hello from Go server 👋")
			})
			d.OnMessage(func(msg webrtc.DataChannelMessage) {
				fmt.Printf("📩 Received: %s\n", string(msg.Data))
			})
		})

		// Offer -Answer
		if err := pc.SetRemoteDescription(offer); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		answer, err := pc.CreateAnswer(nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := pc.SetLocalDescription(answer); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(answer)
	})

	// /candidate endpoint
	http.HandleFunc("/candidate", func(w http.ResponseWriter, r *http.Request) {
		var candidate webrtc.ICECandidateInit
		if err := json.NewDecoder(r.Body).Decode(&candidate); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if pc != nil {
			pc.AddICECandidate(candidate)
		}
	})

	// static
	http.Handle("/", http.FileServer(http.Dir("./static")))

	fmt.Println("🚀 Signaling server started on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
