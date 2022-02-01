package light

import (
	"github.com/markusressel/keyboard-backlight-daemon/internal/util"
	"math"
	"os"
)

const (
	MaxBrightness = "max_brightness"
	Brightness    = "brightness"
)

type Light interface {
	SetBrightness(percentage int) error
	GetBrightness() (percentage int, err error)
}

type light struct {
	path          string
	maxBrightness int
}

func NewLight(path string) Light {

	m, err := util.ReadIntFromFile(path + string(os.PathSeparator) + MaxBrightness)
	if err != nil {
		panic(err)
	}

	return &light{
		path:          path,
		maxBrightness: m,
	}
}

func (f *light) GetBrightness() (percentage int, err error) {
	rawBrightness, err := util.ReadIntFromFile(f.path + string(os.PathSeparator) + Brightness)
	mappedToPercentage := int(math.Round(float64(rawBrightness) / float64(f.maxBrightness)))
	return mappedToPercentage, err
}

func (f *light) SetBrightness(percentage int) (err error) {
	mappedToRange := int(math.Round(float64(percentage) * (float64(f.maxBrightness) / 100)))
	return util.WriteIntToFile(mappedToRange, f.path+string(os.PathSeparator)+Brightness)
}
