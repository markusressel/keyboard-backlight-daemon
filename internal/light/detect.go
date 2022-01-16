package light

import (
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"regexp"
)

const (
	ledsPath = "/sys/class/leds"
)

func DetectKeyboardBacklight() *string {
	files, err := ioutil.ReadDir(ledsPath)
	if err != nil {
		log.Fatal(err)
	}

	r := regexp.MustCompile(".*(kbd|keyboard).*")
	for _, f := range files {
		if r.MatchString(f.Name()) {
			abs, err := filepath.Abs(path.Join(ledsPath, f.Name()))
			if err != nil {
				continue
			}
			return &abs
		}
	}

	return nil
}
