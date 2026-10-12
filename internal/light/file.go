package light

import (
	"fmt"
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

func NewLight(path string) (Light, error) {
	m, err := util.ReadIntFromFile(path + string(os.PathSeparator) + MaxBrightness)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s of %s: %w", MaxBrightness, path, err)
	}
	if m <= 0 {
		return nil, fmt.Errorf("invalid %s of %s: %d", MaxBrightness, path, m)
	}

	return &light{
		path:          path,
		maxBrightness: float64(m),
	}, nil
}

func (f *light) GetBrightness() (percentage float64, err error) {
	rawBrightness, err := util.ReadIntFromFile(f.path + string(os.PathSeparator) + Brightness)
	if err != nil {
		return 0, err
	}
	return float64(rawBrightness) / f.maxBrightness, nil
}

func (f *light) SetBrightness(percentage float64) (err error) {
	mappedToRange := int(math.Round(percentage * f.maxBrightness))
	return util.WriteIntToFile(mappedToRange, f.path+string(os.PathSeparator)+Brightness)
}
