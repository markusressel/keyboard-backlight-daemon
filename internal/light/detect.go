package light

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

const (
	ledsPath = "/sys/class/leds"
)

// DetectKeyboardBacklight returns the path of the first keyboard backlight found,
// or nil if there is none.
func DetectKeyboardBacklight() (*string, error) {
	return detectKeyboardBacklightIn(ledsPath)
}

func detectKeyboardBacklightIn(dir string) (*string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", dir, err)
	}

	r := regexp.MustCompile(".*(kbd|keyboard).*")
	for _, f := range files {
		if r.MatchString(f.Name()) {
			abs, err := filepath.Abs(path.Join(dir, f.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to resolve path of %s: %w", f.Name(), err)
			}
			return &abs, nil
		}
	}

	return nil, nil
}
