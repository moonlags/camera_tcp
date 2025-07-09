package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"unsafe"
)

type Client struct {
	conn   net.Conn
	camera *Camera
}

func newClient(conn net.Conn, c *Camera) Client {
	return Client{
		conn, c,
	}
}

func (c Client) handleConnection() {
	defer c.conn.Close()

	for {
		buf := make([]byte, 256)
		n, err := c.conn.Read(buf)
		if err != nil {
			log.Printf("failed to read from socket %s\n", err)
			break
		}

		log.Printf("recieved message %s\n", buf[:n])

		if (n-len(PASSWORD))%int(unsafe.Sizeof(PhotoConfig{})) != 0 || string(buf[:len(PASSWORD)]) != PASSWORD {
			break
		}

		var photoConfigs PhotoConfig
		reader := bytes.NewReader(buf[len(PASSWORD):n])

		if err := binary.Read(reader, binary.BigEndian, &photoConfigs); err != nil {
			log.Printf("failed to decode binary data %s\n", err)
			break
		}

		log.Printf("%v", photoConfigs)

		if err := c.camera.queuePhotos([]PhotoConfig{photoConfigs}, c.conn); err != nil {
			log.Printf("failed to queue photos %s\n", err)
			break
		}
	}
}
