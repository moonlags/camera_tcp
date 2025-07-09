package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type Photo struct {
	id            uint64
	x             uint16
	y, zoom, mode uint8
	retry         bool
	reciever      net.Conn
	bytes         []byte
}

type PhotoConfig struct {
	x             uint16
	y, zoom, mode uint8
}

func newPhoto(id uint64, c PhotoConfig, reciever net.Conn) (Photo, error) {
	if c.x > 360 {
		return Photo{}, errors.New("X is invalid")
	} else if c.y > 90 {
		return Photo{}, errors.New("Y is invalid")
	} else if c.zoom > 10 {
		return Photo{}, errors.New("ZOOM is invalid")
	} else if c.mode > 13 {
		return Photo{}, errors.New("MODE is invalid")
	}

	return Photo{
		id:       id,
		x:        c.x,
		y:        c.y,
		zoom:     c.zoom,
		mode:     c.mode,
		reciever: reciever,
	}, nil
}

const QUEUE_SIZE = 10

type Camera struct {
	currentPhotoID uint64
	currentX       uint16
	queue          []Photo
}

func newCamera() (*Camera, error) {
	cmd := exec.Command("./motor_driver.bin", "0", "0", "True", "0", "3", "")
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return &Camera{
		queue: make([]Photo, 0),
	}, nil
}

func (c *Camera) queuePhotos(photoConfigs []PhotoConfig, reciever net.Conn) error {
	for _, config := range photoConfigs {
		c.currentPhotoID++

		photo, err := newPhoto(c.currentPhotoID, config, reciever)
		if err != nil {
			reciever.Write([]byte("error"))
			continue
		}

		id := make([]byte, 8)
		binary.BigEndian.PutUint64(id, c.currentPhotoID)
		if _, err := reciever.Write(id); err != nil {
			return err
		}

		if len(c.queue) >= 10 {
			if _, err := reciever.Write(id); err != nil {
				return err
			}
		} else {
			c.queue = append(c.queue, photo)
		}

	}
	return nil
}

func (c *Camera) take() (Photo, error) {
	for len(c.queue) <= 0 {
		time.Sleep(time.Second)
	}

	smallestDistance := math.Abs(float64(c.currentX%180 - c.queue[0].x%180))
	nearestPhotoIndex := 0
	for i, photo := range c.queue {
		dist := math.Abs(float64(c.currentX%180 - photo.x%180))
		if dist < smallestDistance {
			smallestDistance = dist
			nearestPhotoIndex = i
		}
	}

	photo := c.queue[nearestPhotoIndex]
	c.queue = append(c.queue[:nearestPhotoIndex], c.queue[nearestPhotoIndex+1:]...)

	c.setModeAndZoom(photo.mode, photo.zoom)

	cmd := exec.Command("./motor_driver.bin", fmt.Sprint(photo.x), fmt.Sprint(photo.y), "False", fmt.Sprint(c.currentX), "3", "wget -O photoaf.jpg http://127.0.0.1:8080/photoaf.jpg")
	if err := cmd.Run(); err != nil {
		log.Printf("failed to start motor_driver %s\n", err)
		if err := c.phoneInit(); err != nil {
			log.Printf("failed to initialize phone %s\n", err)
		}
		return photo, err
	}

	c.currentX = photo.x

	photoFile, err := os.Open("photoaf.jpg")
	if err != nil {
		return photo, err
	}

	photoBytes, err := io.ReadAll(photoFile)
	if err != nil {
		return photo, err
	}
	photo.bytes = photoBytes

	return photo, nil
}

func (c *Camera) requeuePhoto(p Photo) error {
	if !p.retry {
		p.retry = true
		c.queue = append(c.queue, p)
	} else {
		return errors.New("photo was requeued already")
	}
	return nil
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
