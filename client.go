package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"
	"net"
)

type Client struct {
	conn   net.Conn
	camera *Camera
}

func newClient(conn net.Conn, c *Camera) *Client {
	return &Client{
		conn, c,
	}
}

func (c *Client) handleConnection() {
	defer c.conn.Close()

	reader := bufio.NewReader(c.conn)
	for {
		password := make([]byte, len(PASSWORD))
		if _, err := io.ReadFull(reader, password); err != nil {
			log.Println("failed to read password", err)
			break
		}
		log.Println("recieved password", string(password))

		if string(password) != PASSWORD {
			log.Println("password doesnt match", string(password))
			break
		}

		var config PhotoConfig
		if err := binary.Read(reader, binary.BigEndian, &config); err != nil {
			log.Println("failed to decode photo data", err)
			break
		}

		log.Printf("%v", config)

		out, err := c.camera.queuePhotos(config, c.conn)
		if err != nil {
			log.Println("failed to queue photo", err)

			c.conn.Write([]byte{byte(PhotoError)})
			break
		}

		outData := <-out

		buf := make([]byte, 5)
		binary.Encode(buf, binary.BigEndian, PhotoReady)
		binary.Encode(buf, binary.BigEndian, int32(len(outData)))

		buf = append(buf, outData...)
		if _, err := c.conn.Write(buf); err != nil {
			log.Println("failed to send binary data", err)
			break
		}
	}
}
