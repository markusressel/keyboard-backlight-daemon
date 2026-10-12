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
	return detectKeyboardBacklightIn(ledsPath)
}

func detectKeyboardBacklightIn(dir string) *string {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	r := regexp.MustCompile(".*(kbd|keyboard).*")
	for _, f := range files {
		if r.MatchString(f.Name()) {
			abs, err := filepath.Abs(path.Join(dir, f.Name()))
			if err != nil {
				continue
			}
			return &abs
		}
	}

	return nil
}
