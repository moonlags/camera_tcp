package main

import (
	"encoding/binary"
	"log"
	"net"
	"os"
)

var PASSWORD = os.Getenv("PASSWORD")

func main() {
	if PASSWORD == "" {
		log.Fatal("PASSWORD variable is not set")
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

	log.Printf("started listening on port %s\n", port)

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
		photo, err := c.take()
		if err != nil {
			log.Printf("failed to take photo %s\n", err)

			if err := c.requeuePhoto(photo); err != nil {
				continue
			}

			id := make([]byte, 8)
			binary.BigEndian.PutUint64(id, photo.id)

			photo.reciever.Write(id)
			continue
		}

		sendPhoto(photo)
	}
}

func sendPhoto(p Photo) {
	id := make([]byte, 8)
	binary.BigEndian.PutUint64(id, p.id)

	p.reciever.Write(append(id, p.bytes...))
}
