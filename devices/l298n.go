package devices

import (
	"time"

	"github.com/stianeikeland/go-rpio"
)

// L298N ...
type L298N struct {
	in1 rpio.Pin
	in2 rpio.Pin
	in3 rpio.Pin
	in4 rpio.Pin
	ena rpio.Pin
	enb rpio.Pin
}

// NewL298N ...
func NewL298N(in1, in2, in3, in4, ena, enb uint8) *L298N {
	l := &L298N{
		in1: rpio.Pin(in1),
		in2: rpio.Pin(in2),
		in3: rpio.Pin(in3),
		in4: rpio.Pin(in4),
		ena: rpio.Pin(ena),
		enb: rpio.Pin(enb),
	}
	l.in1.Output()
	l.in2.Output()
	l.in3.Output()
	l.in4.Output()
	l.in1.Low()
	l.in2.Low()
	l.in3.Low()
	l.in4.Low()
	l.ena.Pwm()
	l.enb.Pwm()
	l.speed(90)
	// l.ena.Output()
	// l.enb.Output()
	// l.ena.High()
	// l.enb.High()

	return l
}

// Forward ...
func (l *L298N) Forward() {
	l.in1.High()
	l.in2.Low()
	time.Sleep(70 * time.Millisecond)
	l.in3.High()
	l.in4.Low()

	l.in1.Low()
	time.Sleep(80 * time.Millisecond)
	l.in1.High()
}

// Backward ...
func (l *L298N) Backward() {
	l.in1.Low()
	l.in2.High()
	time.Sleep(70 * time.Millisecond)
	l.in3.Low()
	l.in4.High()

	l.in2.Low()
	time.Sleep(80 * time.Millisecond)
	l.in2.High()
}

// Left ...
func (l *L298N) Left() {
	l.in1.Low()
	l.in2.Low()
	l.in3.High()
	l.in4.Low()
	time.Sleep(100 * time.Millisecond)
}

// Right ...
func (l *L298N) Right() {
	l.in1.High()
	l.in2.Low()
	l.in3.Low()
	l.in4.Low()
	time.Sleep(100 * time.Millisecond)
}

// Stop ...
func (l *L298N) Stop() {
	l.in1.Low()
	l.in2.Low()
	l.in3.Low()
	l.in4.Low()
}

func (l *L298N) speed(n uint32) {
	l.ena.Freq(6400)
	l.enb.Freq(6400)
	l.ena.DutyCycle(n, 100)
	l.enb.DutyCycle(n, 100)
}
