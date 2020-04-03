package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/shanghuiyang/rpi-devices/base"
	"github.com/shanghuiyang/rpi-devices/dev"
	"github.com/stianeikeland/go-rpio"
)

const (
	pinSGH = 12
	pinSGV = 13
	pinLed = 21
	pinBzr = 11
)

const (
	pageOutOfService = `
		<!DOCTYPE html>
		<html>
			<title>Video Monitor</title>
			<h1 style="color:red;font-size:50px;">
				<span style="font-size:100px;">
        			&ensp;&#129318;&#129318;&#129318;<br>
    			</span>
				~~~~~~~~~~~~~~~<br>
				&nbsp;&ensp;Out of Service<br>
				&nbsp;&emsp;20:00 ~ 9:00<br>
				~~~~~~~~~~~~~~~<br>
			</h1>
		</html>
	`
)

var (
	vMonitor    *vmonitor
	pageContext []byte
)

func main() {
	if err := rpio.Open(); err != nil {
		log.Fatalf("failed to open rpio, error: %v", err)
		return
	}
	defer rpio.Close()

	hServo := dev.NewSG90(pinSGH)
	if hServo == nil {
		log.Printf("failed to new a sg90")
		return
	}

	vServo := dev.NewSG90(pinSGV)
	if vServo == nil {
		log.Printf("failed to new a sg90")
		return
	}

	led := dev.NewLed(pinLed)
	if led == nil {
		log.Printf("failed to new a led, will run the monitor without led")
	}

	bzr := dev.NewBuzzer(pinBzr)
	if bzr == nil {
		log.Printf("failed to new a buzzer, will run the monitor without buzzer")
	}

	vMonitor = newVMonitor(hServo, vServo, led, bzr)
	if vMonitor == nil {
		log.Printf("failed to new a vmonitor")
		return
	}

	if err := loadHomePage(); err != nil {
		log.Fatalf("failed to load home page, error: %v", err)
		return
	}

	log.Printf("video monitor server started")
	base.WaitQuit(func() {
		vMonitor.stop()
		rpio.Close()
	})
	vMonitor.start()

	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}

func loadHomePage() error {
	var err error
	pageContext, err = ioutil.ReadFile("vmonitor.html")
	if err != nil {
		return err
	}
	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		homePageHandler(w, r)
	case "POST":
		operationHandler(w, r)
	}
}

func homePageHandler(w http.ResponseWriter, r *http.Request) {
	if vMonitor.outOfService() {
		w.Write([]byte(pageOutOfService))
		return
	}
	w.Write(pageContext)
}

func operationHandler(w http.ResponseWriter, r *http.Request) {
	op := r.FormValue("op")
	switch op {
	case "left":
		go vMonitor.left()
	case "right":
		go vMonitor.right()
	case "up":
		go vMonitor.up()
	case "down":
		go vMonitor.down()
	case "beep":
		go vMonitor.beep(5, 100)
	default:
		log.Printf("invalid operator: %v", op)
	}
}

type vmonitor struct {
	hServo  *dev.SG90
	vServo  *dev.SG90
	led     *dev.Led
	buzzer  *dev.Buzzer
	hAngle  int
	vAngle  int
	chAlert chan int
}

func newVMonitor(hServo, vServo *dev.SG90, led *dev.Led, buzzer *dev.Buzzer) *vmonitor {
	v := &vmonitor{
		hServo:  hServo,
		vServo:  vServo,
		led:     led,
		buzzer:  buzzer,
		hAngle:  0,
		vAngle:  0,
		chAlert: make(chan int, 16),
	}
	return v
}

func (v *vmonitor) start() {
	go v.hServo.Roll(v.hAngle)
	go v.vServo.Roll(v.vAngle)
	go v.alert()
	go v.detectConnecting()
	go v.detectServing()

}

func (v *vmonitor) stop() {
	v.led.Off()
	close(v.chAlert)
}

func (v *vmonitor) left() {
	log.Printf("op: left")
	angle := v.hAngle - 15
	if angle < -90 {
		return
	}
	v.hAngle = angle
	log.Printf("servo: %v", angle)
	v.hServo.Roll(angle)
}

func (v *vmonitor) right() {
	log.Printf("op: right")
	angle := v.hAngle + 15
	if angle > 75 {
		return
	}
	v.hAngle = angle
	log.Printf("servo: %v", angle)
	v.hServo.Roll(angle)
}

func (v *vmonitor) up() {
	log.Printf("op: up")
	angle := v.vAngle + 15
	if angle > 90 {
		return
	}
	v.vAngle = angle
	log.Printf("servo: %v", angle)
	v.vServo.Roll(angle)
}

func (v *vmonitor) down() {
	log.Printf("op: down")
	angle := v.vAngle - 15
	if angle < -30 {
		return
	}
	v.vAngle = angle
	log.Printf("servo: %v", angle)
	v.vServo.Roll(angle)
}

func (v *vmonitor) beep(n int, interval int) {
	log.Printf("op: beep")
	if v.buzzer == nil {
		return
	}
	v.buzzer.Beep(n, interval)
}

func (v *vmonitor) detectConnecting() {
	for {
		time.Sleep(5 * time.Second)
		n, err := v.getConCount()
		if err != nil {
			log.Printf("failed to get users count, err: %v", err)
			continue
		}
		v.chAlert <- n
	}
}

func (v *vmonitor) alert() {
	conCount := 0
	for {
		select {
		case n := <-v.chAlert:
			if n > conCount {
				// there are new connections, give an alert
				go v.beep(2, 100)
			}
			conCount = n
		default:
			// do nothing
		}
		if conCount > 0 {
			v.led.Blink(1, 1000)
		}
		time.Sleep(1 * time.Second)
	}
}

// getConCount is get the count of connecting to the server
func (v *vmonitor) getConCount() (int, error) {
	cmd := `netstat -nat|grep -i "127.0.0.1:8081"|wc -l`
	out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
	if err != nil {
		return 0, err
	}
	var count int
	str := string(out)
	if n, err := fmt.Sscanf(str, "%d\n", &count); n != 1 || err != nil {
		return 0, fmt.Errorf("failed to parse the output of netstat")
	}
	return count, nil
}

func (v *vmonitor) detectServing() {
	inServing := true
	for {
		time.Sleep(1 * time.Minute)
		if v.outOfService() {
			if inServing {
				log.Printf("out of service, stop motion")
				cmd := "sudo killall motion"
				exec.Command("bash", "-c", cmd).CombinedOutput()
				inServing = false
			}
			continue
		}

		if !inServing {
			log.Printf("in service time, start motion")
			cmd := "sudo motion -c /etc/motion/motion.conf"
			exec.Command("bash", "-c", cmd).CombinedOutput()
			inServing = true
		}
	}
}

func (v *vmonitor) outOfService() bool {
	hour := time.Now().Hour()
	if hour >= 20 || hour < 9 {
		// out of service at 20:00~09:00
		return true
	}
	return false
}