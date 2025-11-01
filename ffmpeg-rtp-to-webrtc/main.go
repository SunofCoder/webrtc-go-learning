// SPDX-FileCopyrightText: 2025 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// rtp-to-webrtc demonstrates how to consume an RTP video stream from FFmpeg
// and send it to a WebRTC client using Pion.
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

func main() {
	// Create a new PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		panic(err)
	}

	// Listen for RTP packets from FFmpeg
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5004})
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	// Increase UDP buffer size
	if err := listener.SetReadBuffer(300000); err != nil {
		panic(err)
	}

	// Create a local video track for WebRTC
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion",
	)
	if err != nil {
		panic(err)
	}
	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		panic(err)
	}

	// Read incoming RTCP packets in background
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	// ICE connection state change handler
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State changed: %s\n", state.String())
		if state == webrtc.ICEConnectionStateFailed {
			if closeErr := peerConnection.Close(); closeErr != nil {
				panic(closeErr)
			}
		}
	})

	// Wait for browser SDP offer
	offer := webrtc.SessionDescription{}
	decode(readUntilNewline(), &offer)

	if err = peerConnection.SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	// Create and set local answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	if err = peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}
	<-gatherComplete

	// Output the answer for the browser
	fmt.Println(encode(peerConnection.LocalDescription()))

	// Read RTP packets from FFmpeg and send to WebRTC track
	rtpBuf := make([]byte, 1600)
	packet := &rtp.Packet{}
	for {
		n, _, err := listener.ReadFrom(rtpBuf)
		if err != nil {
			panic(fmt.Sprintf("Error reading RTP packet: %s", err))
		}
		if err = packet.Unmarshal(rtpBuf[:n]); err != nil {
			fmt.Println("Failed to unmarshal RTP packet:", err)
			continue
		}

		if _, err = videoTrack.Write(rtpBuf[:n]); err != nil {
			if errors.Is(err, io.ErrClosedPipe) {
				return
			}
			panic(err)
		}
	}
}

// readUntilNewline reads from stdin until a non-empty line is received
func readUntilNewline() string {
	r := bufio.NewReader(os.Stdin)
	for {
		in, err := r.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			panic(err)
		}
		in = strings.TrimSpace(in)
		if len(in) > 0 {
			return in
		}
	}
}

// encode base64 + JSON encodes the SessionDescription
func encode(desc *webrtc.SessionDescription) string {
	b, err := json.Marshal(desc)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// decode decodes base64 + JSON into a SessionDescription
func decode(in string, desc *webrtc.SessionDescription) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, desc); err != nil {
		panic(err)
	}
}
