package main

import (
	"errors"
	"fmt"
	"httpfromtcp/internal/request"
	"io"
	"log"
	"net"
	"strings"
)

type Config struct {
	Network string
	Port    string
}

var config = Config{
	Network: "tcp",
	Port:    ":42069",
}

func main() {
	listener, err := net.Listen(config.Network, config.Port)
	if err != nil {
		log.Fatalf("could not listen on %s: %s\n", config.Port, err)
	}
	defer listener.Close()

	fmt.Printf("TCP server listening on %s\n", config.Port)
	fmt.Println("=====================================")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("error accepting connection: %s\n", err.Error())
			continue
		}

		fmt.Println("Connection accepted")

		req, err := request.RequestFromReader(conn)
		if err != nil {
			fmt.Printf("error reading request: %s\n", err.Error())
			//conn.Close()
			//continue
		}

		fmt.Println("Request line:")
		fmt.Printf("- Method: %s\n", req.RequestLine.Method)
		fmt.Printf("- Target: %s\n", req.RequestLine.RequestTarget)
		fmt.Printf("- Version: %s\n", req.RequestLine.HttpVersion)
		fmt.Println("Headers:")
		for key, value := range req.Headers {
			fmt.Printf("- %s: %s\n", key, value)
		}

		conn.Close()
		fmt.Println("Connection closed")
	}
}

func getLinesChannel(r io.ReadCloser) <-chan string {
	lines := make(chan string)

	go func() {
		defer r.Close()
		defer close(lines)

		currentLineContents := ""
		for {
			buffer := make([]byte, 8)
			n, err := r.Read(buffer)
			if err != nil {
				if currentLineContents != "" {
					lines <- currentLineContents
					currentLineContents = ""
				}
				if errors.Is(err, io.EOF) {
					break
				}
				fmt.Printf("error: %s\n", err.Error())
				break
			}
			str := string(buffer[:n])
			parts := strings.Split(str, "\n")
			for i := 0; i < len(parts)-1; i++ {
				lines <- currentLineContents + parts[i]
				currentLineContents = ""
			}
			currentLineContents += parts[len(parts)-1]
		}
	}()

	return lines
}

func primeGetLinesChannel(f io.ReadCloser) <-chan string {
	lines := make(chan string)
	go func() {
		defer f.Close()
		defer close(lines)
		currentLineContents := ""
		for {
			b := make([]byte, 8, 8)
			n, err := f.Read(b)
			if err != nil {
				if currentLineContents != "" {
					lines <- currentLineContents
				}
				if errors.Is(err, io.EOF) {
					break
				}
				fmt.Printf("error: %s\n", err.Error())
				return
			}
			str := string(b[:n])
			parts := strings.Split(str, "\n")
			for i := 0; i < len(parts)-1; i++ {
				lines <- fmt.Sprintf("%s%s", currentLineContents, parts[i])
				currentLineContents = ""
			}
			currentLineContents += parts[len(parts)-1]
		}
	}()
	return lines
}
