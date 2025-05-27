package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {

	serverAddress := "localhost:42069"
	udpAddress, err := net.ResolveUDPAddr("udp", serverAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving udp adress: %v\n", err)
		os.Exit(1)
	}

	conn, err := net.DialUDP("udp", nil, udpAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error dialing udp: %v\n", err)
		os.Exit(1)
	}

	defer conn.Close()

	fmt.Println("UDP client started")
	fmt.Println("=====================================")

	fmt.Printf("Sending to %s. Type your message and press Enter to send. Press Ctrl+C to exit.\n", serverAddress)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}

		_, err = conn.Write([]byte(message))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error sending message: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Message sent: %s", message)
	}
}
