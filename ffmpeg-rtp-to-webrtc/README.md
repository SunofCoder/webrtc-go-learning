# FFmpeg RTP to WebRTC Example

This example demonstrates how to consume an RTP video stream from FFmpeg
and send it to a browser using Pion WebRTC.

## Requirements
- Go 1.21+
- FFmpeg

## Run FFmpeg test stream (VP8)
```bash
ffmpeg -re -f lavfi -i testsrc=size=640x480:rate=30 \
  -vcodec libvpx -cpu-used 5 -deadline 1 -g 10 \
  -error-resilient 1 -auto-alt-ref 1 \
  -f rtp rtp://127.0.0.1:5004?pkt_size=1200
