package light

import (
	"github.com/markusressel/keyboard-backlight-daemon/internal/util"
)

type Light interface {
	SetBrightness(percentage float64) error
	GetBrightness() (percentage float64, err error)
}

type light struct {
	path          string
	maxBrightness int
}

func NewLight() Light {
	return &light{
		path:          "",
		maxBrightness: 2,
	}
}

func (f *light) GetBrightness() (percentage float64, err error) {
	rawBrightness, err := util.ReadIntFromFile(f.path)
	mappedToPercentage := float64(rawBrightness) * 33.33333
	return mappedToPercentage, err
}

func (f *light) SetBrightness(percentage float64) (err error) {
	return err
}
