//go:build tinygo && esp32s3_n16r8

package tinygo

import (
	"image/color"
	"machine"
	"time"
)

// boardLED drives the on-board WS2812 RGB LED on GPIO48 using SPI encoding.
type boardLED struct {
	spi   *machine.SPI
	ready bool
	on    bool
	cur   color.RGBA
	stop  chan struct{}
}

func newLED() LED {
	led := &boardLED{
		spi: machine.SPI0,
		cur: color.RGBA{R: 0x00, G: 0xff, B: 0x00},
	}
	if err := led.spi.Configure(machine.SPIConfig{
		Frequency: 3_200_000,
		Mode:      0,
		SCK:       machine.NoPin,
		SDO:       machine.GPIO48,
		SDI:       machine.NoPin,
	}); err == nil {
		led.ready = true
	}
	return led
}

func (l *boardLED) On() {
	l.on = true
	l.writeColor(l.cur)
}

func (l *boardLED) Off() {
	l.on = false
	l.writeColor(color.RGBA{R: 0x00, G: 0x00, B: 0x00})
}

func (l *boardLED) Toggle() {
	if l.on {
		l.Off()
		return
	}
	l.On()
}

func (l *boardLED) Red() {
	l.RGB(0xff, 0x00, 0x00)
}

func (l *boardLED) Green() {
	l.RGB(0x00, 0xff, 0x00)
}

func (l *boardLED) Blue() {
	l.RGB(0x00, 0x00, 0xff)
}

func (l *boardLED) White() {
	l.RGB(0xff, 0xff, 0xff)
}

func (l *boardLED) RGB(r, g, b uint8) {
	l.cur = color.RGBA{R: r, G: g, B: b}
	l.on = true
	l.writeColor(l.cur)
}

func (l *boardLED) writeColor(c color.RGBA) {
	if !l.ready {
		return
	}
	var buf [14]byte
	ws2812EncodeByte(c.G, buf[0:4])
	ws2812EncodeByte(c.R, buf[4:8])
	ws2812EncodeByte(c.B, buf[8:12])
	_ = l.spi.Tx(buf[:], nil)
}

func ws2812EncodeByte(value uint8, out []byte) {
	for i := 0; i < 4; i++ {
		hi := (value >> (7 - uint(i)*2)) & 1
		lo := (value >> (6 - uint(i)*2)) & 1
		var encoded byte
		if hi != 0 {
			encoded |= 0xE0
		} else {
			encoded |= 0x80
		}
		if lo != 0 {
			encoded |= 0x0E
		} else {
			encoded |= 0x08
		}
		out[i] = encoded
	}
}

func (l *boardLED) BlinkEvery(ms int) {
	if l.stop != nil {
		close(l.stop)
	}
	stop := make(chan struct{})
	l.stop = stop
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-time.After(time.Duration(ms) * time.Millisecond):
				l.Toggle()
			}
		}
	}()
}

func (l *boardLED) StopBlink() {
	if l.stop != nil {
		close(l.stop)
		l.stop = nil
	}
}
