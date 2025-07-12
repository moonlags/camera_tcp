package main

import (
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

	for {
		password := make([]byte, len(PASSWORD))
		if _, err := io.ReadFull(c.conn, password); err != nil {
			log.Println("failed to read password", err)
			break
		}
		log.Println("recieved password", string(password), len(password))

		if string(password) != PASSWORD {
			log.Println("password doesnt match", string(password))
			break
		}

		var config PhotoConfig
		if err := binary.Read(c.conn, binary.BigEndian, &config); err != nil {
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

		if _, err := c.conn.Write([]byte{byte(PhotoReady)}); err != nil {
			log.Println("failed to write message code", err)
			break
		}

		if err := binary.Write(c.conn, binary.BigEndian, int32(len(outData))); err != nil {
			log.Println("failed to write outData lenght", err)
			break
		}

		if _, err := c.conn.Write(outData); err != nil {
			log.Println("failed to write outData", err)
			break
		}
	}
}
