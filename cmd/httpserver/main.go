package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"httpfromtcp/internal/headers"
	"httpfromtcp/internal/request"
	"httpfromtcp/internal/response"
	"httpfromtcp/internal/server"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const port = 42069

func handler(w *response.Writer, req *request.Request) {
	target := req.RequestLine.RequestTarget

	// Check if this is a proxy request
	if strings.HasPrefix(target, "/httpbin") {
		handleProxy(w, target)
		return
	}

	// Handle video endpoint
	if target == "/video" {
		handleVideo(w)
		return
	}

	var statusCode response.StatusCode
	var htmlBody string

	if target == "/yourproblem" {
		statusCode = response.StatusBadRequest
		htmlBody = `<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
  </body>
</html>`
	} else if target == "/myproblem" {
		statusCode = response.StatusInternalServerError
		htmlBody = `<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
  </body>
</html>`
	} else {
		statusCode = response.StatusOK
		htmlBody = `<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>`
	}

	// Write status line
	err := w.WriteStatusLine(statusCode)
	if err != nil {
		return
	}

	// Create headers with Content-Type set to text/html
	headers := headers.NewHeaders()
	headers.Set("Content-Length", fmt.Sprintf("%d", len(htmlBody)))
	headers.Set("Connection", "close")
	headers.SetOverride("Content-Type", "text/html")

	// Write headers
	err = w.WriteHeaders(headers)
	if err != nil {
		return
	}

	// Write body
	_, err = w.WriteBody([]byte(htmlBody))
	if err != nil {
		return
	}
}

func handleVideo(w *response.Writer) {
	// Read the video file
	videoData, err := os.ReadFile("assets/vim.mp4")
	if err != nil {
		// Write error response
		w.WriteStatusLine(response.StatusInternalServerError)
		errorHeaders := headers.NewHeaders()
		errorHeaders.Set("Connection", "close")
		w.WriteHeaders(errorHeaders)
		w.WriteBody([]byte("Error reading video file: " + err.Error()))
		return
	}

	// Write status line
	err = w.WriteStatusLine(response.StatusOK)
	if err != nil {
		return
	}

	// Create headers with Content-Type set to video/mp4
	videoHeaders := headers.NewHeaders()
	videoHeaders.Set("Content-Length", fmt.Sprintf("%d", len(videoData)))
	videoHeaders.Set("Connection", "close")
	videoHeaders.SetOverride("Content-Type", "video/mp4")

	// Write headers
	err = w.WriteHeaders(videoHeaders)
	if err != nil {
		return
	}

	// Write body
	_, err = w.WriteBody(videoData)
	if err != nil {
		return
	}
}

func handleProxy(w *response.Writer, target string) {
	// Extract the path after /httpbin
	path := strings.TrimPrefix(target, "/httpbin")
	if path == "" {
		path = "/"
	}

	// Make request to httpbin.org
	url := "https://httpbin.org" + path
	resp, err := http.Get(url)
	if err != nil {
		// Write error response
		w.WriteStatusLine(response.StatusInternalServerError)
		errorHeaders := headers.NewHeaders()
		errorHeaders.Set("Connection", "close")
		w.WriteHeaders(errorHeaders)
		w.WriteBody([]byte("Proxy error: " + err.Error()))
		return
	}
	defer resp.Body.Close()

	// Convert HTTP status code to our StatusCode type
	statusCode := response.StatusCode(resp.StatusCode)

	// Write status line
	err = w.WriteStatusLine(statusCode)
	if err != nil {
		return
	}

	// Copy headers from httpbin response, but remove Content-Length and add Transfer-Encoding: chunked
	proxyHeaders := headers.NewHeaders()
	for key, values := range resp.Header {
		lowerKey := strings.ToLower(key)
		// Skip Content-Length header
		if lowerKey == "content-length" {
			continue
		}
		// Copy all other headers
		for _, value := range values {
			proxyHeaders.Set(key, value)
		}
	}

	// Add Transfer-Encoding: chunked
	proxyHeaders.SetOverride("Transfer-Encoding", "chunked")
	// Announce trailers
	proxyHeaders.Set("Trailer", "X-Content-SHA256, X-Content-Length")
	proxyHeaders.Set("Connection", "close")

	// Write headers
	err = w.WriteHeaders(proxyHeaders)
	if err != nil {
		return
	}

	// Read response body in chunks, track full body, and forward them
	buffer := make([]byte, 1024)
	var fullBody []byte
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			// Track the full body
			fullBody = append(fullBody, buffer[:n]...)
			// Write chunk
			_, writeErr := w.WriteChunkedBody(buffer[:n])
			if writeErr != nil {
				return
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}
	}

	// Write final chunk marker
	_, err = w.WriteChunkedBodyDone()
	if err != nil {
		return
	}

	// Calculate SHA256 hash of the full body
	hash := sha256.Sum256(fullBody)
	hashHex := hex.EncodeToString(hash[:])

	// Create trailers
	trailers := headers.NewHeaders()
	trailers.Set("X-Content-SHA256", hashHex)
	trailers.Set("X-Content-Length", fmt.Sprintf("%d", len(fullBody)))

	// Write trailers
	err = w.WriteTrailers(trailers)
	if err != nil {
		return
	}
}

func main() {
	server, err := server.Serve(port, handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
