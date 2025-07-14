package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"time"
)

type Photo struct {
	x             uint16
	y, zoom, mode uint8
	retry         bool
	output        chan []byte
}

type PhotoConfig struct {
	X             uint16
	Y, Zoom, Mode uint8
}

func newPhoto(c PhotoConfig, out chan []byte) (Photo, error) {
	if c.X > 360 {
		return Photo{}, errors.New("X is invalid")
	} else if c.Y > 90 {
		return Photo{}, errors.New("Y is invalid")
	} else if c.Zoom > 10 {
		return Photo{}, errors.New("ZOOM is invalid")
	} else if c.Mode > 13 {
		return Photo{}, errors.New("MODE is invalid")
	}

	return Photo{
		x:      c.X,
		y:      c.Y,
		zoom:   c.Zoom,
		mode:   c.Mode,
		output: out,
	}, nil
}

const QUEUE_SIZE = 10

type Camera struct {
	currentX  uint16
	queue     chan Photo
	turnedOff bool
}

func newCamera() (*Camera, error) {
	if err := sendCommand(0, 0, 1, 1); err != nil {
		return nil, err
	}

	return &Camera{
		queue:     make(chan Photo, QUEUE_SIZE),
		turnedOff: true,
	}, nil
}

func (c *Camera) queuePhotos(config PhotoConfig) (chan []byte, error) {
	out := make(chan []byte)

	photo, err := newPhoto(config, out)
	if err != nil {
		return nil, err
	}

	if len(c.queue) >= QUEUE_SIZE {
		return nil, err
	}
	c.queue <- photo

	return out, nil
}

func (c *Camera) take(p Photo) ([]byte, error) {
	c.setModeAndZoom(p.mode, p.zoom)

	if err := sendCommand(p.x, p.y, 0, 0); err != nil {
		return nil, err
	}
	c.turnedOff = false
	c.currentX = p.x

	resp, err := http.DefaultClient.Get("http://127.0.0.1:8080/photoaf.jpg")
	if err != nil {
		log.Println("failed to request photo", err)
		if err2 := c.phoneInit(); err2 != nil {
			log.Printf("failed to initialize phone %s\n", err2)
		}
		return nil, err
	}
	defer resp.Body.Close()

	photoBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return photoBytes, nil
}

func (c *Camera) requeuePhoto(p Photo) error {
	if !p.retry {
		p.retry = true
		c.queue <- p
		return nil

	}
	return errors.New("photo was requeued already")
}

func (c Camera) setModeAndZoom(mode uint8, zoom uint8) error {
	modes := []string{"none", "mono", "negative", "sepia", "aqua", "whiteboard", "blackboard", "nashville", "hefe", "valencia", "xproll", "lofi", "sierra", "walden"}
	url := "http://127.0.0.1:8080/settings/coloreffect?set=" + modes[mode]
	if _, err := http.Get(url); err != nil {
		return err
	}

	url = fmt.Sprintf("http://127.0.0.1:8080/ptz?zoom=%d", zoom)
	if _, err := http.Get(url); err != nil {
		return err
	}
	return nil
}

func (c Camera) phoneInit() error {
	if err := exec.Command("./phone_init.sh").Run(); err != nil {
		return err
	}
	return nil
}

func sendCommand(x uint16, y, init, motorOff uint8) error {
	conn, err := net.Dial("tcp", DRIVER_ADDRESS)
	if err != nil {
		return err
	}
	defer conn.Close()

	msg := fmt.Sprintf("%d %d %d %d", x, y, init, motorOff)
	_, err = conn.Write([]byte(msg))
	if err != nil {
		return err
	}

	buf := make([]byte, 128)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}

	log.Println("message from driver", string(buf[:n]))
	time.Sleep(time.Second * 3)

	return nil
}
