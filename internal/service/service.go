package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/markusressel/keyboard-backlight-daemon/internal/config"
	"github.com/markusressel/keyboard-backlight-daemon/internal/light"
	"github.com/markusressel/keyboard-backlight-daemon/internal/util"
	"github.com/oklog/run"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"
)

type Event struct {
	Sec  uint64
	Usec uint64
	Type uint16
	Code uint16
}

var (
	inputEventChannel = make(chan Event, 1)
	userIdleChannel   = make(chan bool)
)

type KbdService struct {
	initialized bool
	light       light.Light

	lastNonIdleBrightness int

	userIdle      bool
	userIdleTimer *time.Timer
	idleTimeout   time.Duration

	currentlyWatchedInputDevices map[string]bool
	animationTarget              *AnimationTarget
}

func NewKbdService(c config.Configuration, l light.Light) *KbdService {
	return &KbdService{
		light:                        l,
		idleTimeout:                  c.IdleTimeout,
		userIdle:                     true,
		currentlyWatchedInputDevices: map[string]bool{},
	}
}

func (s *KbdService) Run() {

	b, err := s.light.GetBrightness()
	if err != nil {
		panic(err)
	}
	s.lastNonIdleBrightness = b

	ctx, cancel := context.WithCancel(context.Background())

	var g run.Group
	{
		g.Add(func() error {
			return s.watchInputDevices(ctx)
		}, func(err error) {
			if err != nil {
				fmt.Printf("Error: %v", err)
			}
		})
	}
	{
		g.Add(func() error {
			totalAnimationTimeOn := config.CurrentConfig.AnimationTimeOn
			totalAnimationTimeOff := config.CurrentConfig.AnimationTimeOff

			s.animationTarget = &AnimationTarget{
				when: time.Now(),
				from: s.lastNonIdleBrightness,
				to:   s.lastNonIdleBrightness,
			}

			frameTicker := time.Tick(100 * time.Millisecond)
			lastSetPercentage := s.lastNonIdleBrightness

			durationSinceStart := 0 * time.Second

			for {
				select {
				case <-ctx.Done():
					return nil
				case <-frameTicker:
					var totalAnimationTime time.Duration
					if s.animationTarget.to == lastSetPercentage {
						continue
					} else if s.animationTarget.to > lastSetPercentage {
						totalAnimationTime = totalAnimationTimeOn
					} else {
						totalAnimationTime = totalAnimationTimeOff
					}

					durationSinceStart = time.Now().Sub(s.animationTarget.when)
					progress := float64(durationSinceStart.Milliseconds()) / float64(totalAnimationTime.Milliseconds())
					diff := s.animationTarget.to - s.animationTarget.from
					animatedValue := s.animationTarget.from + int(float64(diff)*progress)
					targetValue := int(math.Min(float64(animatedValue), 100))
					targetValue = int(math.Max(float64(targetValue), 0))

					if targetValue == lastSetPercentage {
						// nothing to do
						continue
					}

					fmt.Printf("Setting brightness from %d to %d -> %d\n", lastSetPercentage, targetValue, s.animationTarget.to)
					err = s.light.SetBrightness(targetValue)
					if err != nil {
						fmt.Printf("%v", err)
					}
					lastSetPercentage = targetValue
				}
			}
		}, func(err error) {
			if err != nil {
				fmt.Printf("Error: %v", err)
			}
		})
	}
	{
		g.Add(func() error {
			return s.controlLoop(ctx)
		}, func(err error) {
			if err != nil {
				fmt.Printf("Error: %v", err)
			}
		})
	}
	{
		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM, os.Kill)

		g.Add(func() error {
			for {
				s := <-sig
				if s != nil {
					fmt.Printf("Received '%s' signal, exiting...", s)
					os.Exit(0)
				}
			}
		}, func(err error) {
			cancel()
			close(sig)
		})
	}

	if err := g.Run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

func (s *KbdService) controlLoop(ctx context.Context) error {
	s.userIdleTimer = time.AfterFunc(s.idleTimeout, func() {
		userIdleChannel <- true
	})

	for {
		select {
		case <-ctx.Done():
			// (try to) restore brightness on exit
			s.light.SetBrightness(s.lastNonIdleBrightness)
			return nil
		case isActive := <-userIdleChannel:
			s.updateState(isActive)
		case <-inputEventChannel:
			s.onUserInteraction()
		}
	}
}

func (s *KbdService) onUserInteraction() {
	s.userIdleTimer.Reset(s.idleTimeout)
	go func() {
		userIdleChannel <- false
	}()
}

func (s *KbdService) updateState(userIdle bool) {
	if s.initialized == false {
		s.initialized = true
	} else if s.userIdle == userIdle {
		return
	}

	s.userIdle = userIdle
	// TODO: verbose
	//fmt.Printf("UserIdle: %t\n", userIdle)

	b, err := s.light.GetBrightness()
	if err != nil {
		return
	}

	if userIdle {
		// update the target brightness to the
		// last brightness before detecting "idle" state
		if err == nil && b != s.lastNonIdleBrightness {
			fmt.Printf("Updating target brightness: %d -> %d\n", s.lastNonIdleBrightness, b)
			s.lastNonIdleBrightness = b
		}
		s.animateBrightness(b, 0)
	} else {
		s.animateBrightness(b, s.lastNonIdleBrightness)
	}
}

type AnimationTarget struct {
	when time.Time
	from int
	to   int
}

// animateBrightness animates the brightness of the keyboard backlight to the given
// currentTargetBrightness
func (s *KbdService) animateBrightness(from int, percentage int) {
	s.animationTarget = &AnimationTarget{
		when: time.Now(),
		from: from,
		to:   percentage,
	}
}

// listenToEvents listens to incoming events on the given file
// and notifies the given channel ch for each one.
func (s *KbdService) listenToEvents(path string, ch chan Event) error {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	b := make([]byte, 24)

	for {
		_, err = f.Read(b)
		if err != nil {
			return err
		}
		event := Event{
			Sec:  binary.LittleEndian.Uint64(b[0:8]),
			Usec: binary.LittleEndian.Uint64(b[8:16]),
			Type: binary.LittleEndian.Uint16(b[16:18]),
			Code: binary.LittleEndian.Uint16(b[18:20]),
		}
		var value int32
		err = binary.Read(bytes.NewReader(b[20:]), binary.LittleEndian, &value)
		if err != nil {
			continue
		}
		go func() {
			ch <- event
		}()
	}
}

func (s *KbdService) watchInputDevices(ctx context.Context) error {

	staticPaths := []string{}
	// from configuration
	for _, path := range config.CurrentConfig.InputEventDevices {
		staticPaths = append(staticPaths, path)
	}

	micePath := "/dev/input/mice"
	staticPaths = append(staticPaths, micePath)

	s.scanInputDevices(staticPaths)
	tick := time.Tick(10 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick:
			s.scanInputDevices(staticPaths)
		}
	}
}

var kbdPattern = regexp.MustCompile(".*kbd.*")

func (s *KbdService) scanInputDevices(staticPaths []string) {
	paths := []string{}
	for _, path := range staticPaths {
		paths = append(paths, path)
	}

	matches := util.FindFilesMatching("/dev/input/by-id", kbdPattern)
	for _, match := range matches {
		paths = append(paths, match)
	}

	for _, p := range paths {
		path, err := filepath.EvalSymlinks(p)
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}

		v, ok := s.currentlyWatchedInputDevices[path]
		if ok && v == true {
			continue
		}

		fmt.Printf("Listening to: %s\n", path)
		s.currentlyWatchedInputDevices[path] = true

		go func() {
			defer func() { s.currentlyWatchedInputDevices[path] = false }()
			s.listenToEvents(path, inputEventChannel)
		}()
	}
}
