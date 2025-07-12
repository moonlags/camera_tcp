package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
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
	currentPhotoID uint64
	currentX       uint16
	queue          chan Photo
}

func newCamera() (*Camera, error) {
	cmd := exec.Command("./motor_driver.bin", "0", "0", "True", "0", "3", "")
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return &Camera{
		queue: make(chan Photo, QUEUE_SIZE),
	}, nil
}

func (c *Camera) queuePhotos(config PhotoConfig, reciever net.Conn) (chan []byte, error) {
	out := make(chan []byte)

	c.currentPhotoID++

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

	cmd := exec.Command("./motor_driver.bin", fmt.Sprint(p.x), fmt.Sprint(p.y), "False", fmt.Sprint(c.currentX), "3", "wget -O photoaf.jpg http://127.0.0.1:8080/photoaf.jpg")
	if err := cmd.Run(); err != nil {
		log.Printf("failed to start motor_driver %s\n", err)
		if err := c.phoneInit(); err != nil {
			log.Printf("failed to initialize phone %s\n", err)
		}
		return nil, err
	}

	c.currentX = p.x

	photoFile, err := os.Open("photoaf.jpg")
	if err != nil {
		return nil, err
	}
	defer photoFile.Close()

	photoBytes, err := io.ReadAll(photoFile)
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
