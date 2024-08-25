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
	// SetBrightness sets the brightness of this light as a percentage in range [0..1]
	SetBrightness(percentage float64) error
	// GetBrightness returns the brightness of this light as a percentage in range [0..1]
	GetBrightness() (percentage float64, err error)
}

type light struct {
	path          string
	maxBrightness float64
}

func NewLight(path string) Light {

	m, err := util.ReadIntFromFile(path + string(os.PathSeparator) + MaxBrightness)
	if err != nil {
		panic(err)
	}

	return &light{
		path:          path,
		maxBrightness: float64(m),
	}
}

func (f *light) GetBrightness() (percentage float64, err error) {
	rawBrightness, err := util.ReadIntFromFile(f.path + string(os.PathSeparator) + Brightness)
	mappedToPercentage := math.Round(float64(rawBrightness) / float64(f.maxBrightness))
	return mappedToPercentage, err
}

func (f *light) SetBrightness(percentage float64) (err error) {
	mappedToRange := int(math.Round((percentage * 100) * (float64(f.maxBrightness) / 100)))
	return util.WriteIntToFile(mappedToRange, f.path+string(os.PathSeparator)+Brightness)
}
