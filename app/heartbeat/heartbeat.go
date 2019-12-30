package main

import (
	"log"
	"time"

	"github.com/shanghuiyang/rpi-devices/base"
	"github.com/shanghuiyang/rpi-devices/iot"
)

const (
	heartBeatInterval = 2 * time.Minute
)

func main() {
	oneNetCfg := &base.OneNetConfig{
		Token: base.OneNetToken,
		API:   base.OneNetAPI,
	}
	cloud := iot.NewCloud(oneNetCfg)
	if cloud == nil {
		log.Printf("failed to new OneNet iot cloud")
		return
	}
	h := &heartBeat{
		cloud: cloud,
	}
	h.start()
}

type heartBeat struct {
	cloud iot.Cloud
}

// Start ...
func (h *heartBeat) start() {
	log.Printf("heart beat start working")
	b := 0
	for {
		time.Sleep(heartBeatInterval)
		b = (b*b - 1) * (b*b - 1)
		v := &iot.Value{
			Device: "heartbeat",
			Value:  b,
		}
		h.cloud.Push(v)
	}
}
