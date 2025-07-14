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

const (
	PORT           = ":54321"
	DRIVER_ADDRESS = "127.0.0.1:8000"
)

var PASSWORD = os.Getenv("CAMERA_PASSWORD")

func main() {
	if PASSWORD == "" {
		log.Fatal("CAMERA_PASSWORD variable is not set")
	}

	listener, err := net.Listen("tcp", PORT)
	if err != nil {
		log.Fatalf("failed to start listening on port %s: %v", PORT, err)
	}
	defer listener.Close()

	log.Println("started listening on port", PORT)

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
			if !c.turnedOff {
				sendCommand(c.currentX, 0, 0, 1)
				c.turnedOff = true
			}
		}
	}
}
