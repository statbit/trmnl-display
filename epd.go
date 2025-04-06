package main

import (
	"fmt"
	"image"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

// EPD represents the E-Ink display
type EPD struct {
	width   int
	height  int
	rstPin  rpio.Pin
	dcPin   rpio.Pin
	csPin   rpio.Pin
	busyPin rpio.Pin
}

// NewEPD creates a new EPD instance
func NewEPD() (*EPD, error) {
	// Initialize GPIO
	if err := rpio.Open(); err != nil {
		return nil, fmt.Errorf("failed to initialize GPIO: %v", err)
	}

	epd := &EPD{
		width:   640,
		height:  384,
		rstPin:  rpio.Pin(17), // GPIO17
		dcPin:   rpio.Pin(25), // GPIO25
		csPin:   rpio.Pin(8),  // GPIO8 (CE0)
		busyPin: rpio.Pin(24), // GPIO24
	}

	// Set pin modes
	epd.rstPin.Output()
	epd.dcPin.Output()
	epd.csPin.Output()
	epd.busyPin.Input()

	// Initialize SPI
	if err := rpio.SpiBegin(rpio.Spi0); err != nil {
		return nil, fmt.Errorf("failed to initialize SPI: %v", err)
	}
	rpio.SpiSpeed(4000000) // 4MHz
	rpio.SpiMode(0, 0)

	return epd, nil
}

// Close cleans up resources
func (e *EPD) Close() {
	rpio.SpiEnd(rpio.Spi0)
	rpio.Close()
}

// Reset the display
func (e *EPD) Reset() {
	e.rstPin.High()
	time.Sleep(200 * time.Millisecond)
	e.rstPin.Low()
	time.Sleep(200 * time.Millisecond)
	e.rstPin.High()
	time.Sleep(200 * time.Millisecond)
}

// WaitUntilIdle waits until the display is not busy
func (e *EPD) WaitUntilIdle() {
	for e.busyPin.Read() == rpio.High {
		time.Sleep(100 * time.Millisecond)
	}
}

// SendCommand sends a command to the display
func (e *EPD) SendCommand(command byte) {
	e.dcPin.Low()
	e.csPin.Low()
	rpio.SpiTransmit(command)
	e.csPin.High()
}

// SendData sends data to the display
func (e *EPD) SendData(data byte) {
	e.dcPin.High()
	e.csPin.Low()
	rpio.SpiTransmit(data)
	e.csPin.High()
}

// Init initializes the display
func (e *EPD) Init() {
	e.Reset()

	// Power settings
	e.SendCommand(0x01)
	e.SendData(0x37)
	e.SendData(0x00)

	// Power on
	e.SendCommand(0x04)
	e.WaitUntilIdle()

	// Panel setting
	e.SendCommand(0x00)
	e.SendData(0xCF)
	e.SendData(0x08)

	// Resolution setting
	e.SendCommand(0x61)
	e.SendData(0x02) // 640
	e.SendData(0x80)
	e.SendData(0x01) // 384
	e.SendData(0x80)

	// VCOM and data interval setting
	e.SendCommand(0x50)
	e.SendData(0x77)

	// TCON setting
	e.SendCommand(0x60)
	e.SendData(0x22)

	// PLL control
	e.SendCommand(0x30)
	e.SendData(0x3C)

	// Power off sequence
	e.SendCommand(0x02)
	e.WaitUntilIdle()
}

// Clear clears the display
func (e *EPD) Clear() {
	e.SendCommand(0x10)
	for i := 0; i < e.width*e.height/8; i++ {
		e.SendData(0xFF)
	}

	e.SendCommand(0x13)
	for i := 0; i < e.width*e.height/8; i++ {
		e.SendData(0xFF)
	}

	e.Display()
}

// Display updates the display with the current image
func (e *EPD) Display() {
	e.SendCommand(0x12)
	time.Sleep(100 * time.Millisecond)
	e.WaitUntilIdle()
}

// DisplayImage displays an image on the E-Ink display
func (e *EPD) DisplayImage(img image.Image) error {
	// Convert image to black and white
	bounds := img.Bounds()
	if bounds.Dx() != e.width || bounds.Dy() != e.height {
		return fmt.Errorf("image dimensions (%dx%d) don't match display (%dx%d)",
			bounds.Dx(), bounds.Dy(), e.width, e.height)
	}

	// Send black data
	e.SendCommand(0x10)
	for y := 0; y < e.height; y++ {
		for x := 0; x < e.width; x++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			// Convert to black and white
			if (r+g+b)/3 > 0x7FFF {
				e.SendData(0xFF) // White
			} else {
				e.SendData(0x00) // Black
			}
		}
	}

	// Send red data (if supported by your display)
	e.SendCommand(0x13)
	for y := 0; y < e.height; y++ {
		for x := 0; x < e.width; x++ {
			e.SendData(0x00) // No red
		}
	}

	e.Display()
	return nil
}
