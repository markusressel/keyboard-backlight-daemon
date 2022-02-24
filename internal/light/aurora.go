package light

import (
	"github.com/karalabe/usb"
	"math"
)

type auroraLight struct {
	device         usb.DeviceInfo
	maxBrightness  int
	lastBrightness *int
}

func NewAuroraLight() Light {

	device := findUsbDevice()

	return &auroraLight{
		device:        device,
		maxBrightness: 3,
	}
}

func (l *auroraLight) GetBrightness() (percentage int, err error) {
	if l.lastBrightness != nil {
		percentage = *l.lastBrightness
	} else {
		percentage = 100
	}

	return percentage, err
}

func (l *auroraLight) SetBrightness(percentage int) (err error) {
	maxAlpha := 0xff
	mappedToRange := uint8(math.Round(float64(percentage) * (float64(maxAlpha) / 100)))

	c := Color{mappedToRange, 0x00, 0x00}
	err = l.sendMessages(true, singleStatic(c))

	if err == nil {
		l.lastBrightness = &percentage
	}
	return err
}

func (l *auroraLight) SetColor(a uint8, r uint8, g uint8, b uint8) (err error) {
	c := Color{red: r, green: g, blue: b}
	return l.sendMessages(true, singleStatic(c))
}

type Color struct {
	red   uint8
	green uint8
	blue  uint8
}

type message [17]uint8

var (
	MessageOther      = message{0x5d, 0xb3}
	MessageBrightness = message{0x5a, 0xba, 0xc5, 0xc4}
	MessageSet        = message{0x5d, 0xb5}
	MessageApply      = message{0x5d, 0xb4}
	MessageInitialize = message{0x5a, 0x41, 0x53, 0x55, 0x53, 0x20, 0x54, 0x65, 0x63, 0x68, 0x2e, 0x49, 0x6e, 0x63, 0x2e, 0x00}
)

func singleStatic(color Color) message {
	m := newMessage(MessageOther)
	m[4] = color.red
	m[5] = color.green
	m[6] = color.blue
	return m
}

func setBrightness(scalar uint8) message {
	m := newMessage(MessageBrightness)
	m[4] = scalar
	return m
}

func newMessage(base message) message {
	m := message{}
	for i, u := range base {
		m[i] = u
	}
	return m
}

func findUsbDevice() usb.DeviceInfo {
	const AsusVendorId uint16 = 0x0b05
	//var AsusProductIds = []uint16{0x1854, 0x1869, 0x1866, 0x19b6}

	// Enumerate all the HID devices in alphabetical path order

	hids, err := usb.EnumerateHid(AsusVendorId, 0)
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(hids); i++ {
		for j := i + 1; j < len(hids); j++ {
			if hids[i].Path > hids[j].Path {
				hids[i], hids[j] = hids[j], hids[i]
			}
		}
	}

	return hids[0]
}

func (l *auroraLight) sendMessages(setAndApply bool, messages ...message) error {
	if setAndApply {
		messages = append(messages, MessageSet, MessageApply)
	}

	device, err := l.device.Open()
	defer device.Close()
	if err != nil {
		return err
	}

	err = sendMessages(device, MessageInitialize)
	return sendMessages(device, messages...)
}

func sendMessages(device usb.Device, messages ...message) (err error) {
	for _, message := range messages {
		buf := make([]byte, len(message))
		for i, u := range message {
			buf[i] = u
		}

		//fmt.Printf("%v\n", buf)
		if _, err = device.Write(buf); err != nil {
			break
		}
	}

	return err
}
