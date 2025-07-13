package main

import (
	"log"
	"net"
	"os"
	"time"
)

const (
	PhotoError uint8 = iota
	PhotoReady
)

var PASSWORD = os.Getenv("CAMERA_PASSWORD")

func main() {
	if PASSWORD == "" {
		log.Fatal("CAMERA_PASSWORD variable is not set")
	}

	port := os.Getenv("TCP_PORT")
	if port == "" {
		log.Fatal("TCP_PORT variable is not set")
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to start listening on port %s: %v", port, err)
	}
	defer listener.Close()

	log.Println("started listening on port", port)

	camera, err := newCamera()
	if err != nil {
		log.Fatalf("failed to initialize camera %v", err)
	}

	go photoHandler(camera)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("failed to accept connection %s\n", err)
			continue
		}

		client := newClient(conn, camera)
		go client.handleConnection()
	}
}

func photoHandler(c *Camera) {
	for {
		select {
		case photo := <-c.queue:
			data, err := c.take(photo)
			if err != nil {
				log.Printf("failed to take photo %s\n", err)

				if err := c.requeuePhoto(photo); err == nil {
					continue
				}
			}
			photo.output <- data
		case <-time.After(time.Minute):
			sendCommand(c.currentX, 0, 0, 1)
		}
	}
}
