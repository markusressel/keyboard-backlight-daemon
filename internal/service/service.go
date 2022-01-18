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
	initialized      bool
	idleTimeout      time.Duration
	light            light.Light
	targetBrightness int
	userIdle         bool
	userIdleTimer    *time.Timer
}

func NewKbdService(c config.Configuration, l light.Light) *KbdService {
	return &KbdService{
		light:       l,
		idleTimeout: c.IdleTimeout,
		userIdle:    true,
	}
}

func (s *KbdService) Run() {

	b, err := s.light.GetBrightness()
	if err != nil {
		panic(err)
	}
	s.targetBrightness = b

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

func (s KbdService) controlLoop(ctx context.Context) error {
	s.userIdleTimer = time.AfterFunc(s.idleTimeout, func() {
		userIdleChannel <- true
	})

	for {
		select {
		case <-ctx.Done():
			// (try to) restore brightness on exit
			s.light.SetBrightness(s.targetBrightness)
			return nil
		case isActive := <-userIdleChannel:
			s.updateState(isActive)
		case <-inputEventChannel:
			s.onUserInteraction()
		}
	}
}

func (s KbdService) onUserInteraction() {
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

	if userIdle {
		// update the target brightness to the
		// last brightness before detecting "idle" state
		b, err := s.light.GetBrightness()
		if err == nil && b != s.targetBrightness {
			fmt.Printf("Updating target brightness: %d -> %d\n", s.targetBrightness, b)
			s.targetBrightness = b
		}
		s.light.SetBrightness(0)
	} else {
		s.light.SetBrightness(s.targetBrightness)
	}
}

// listenToEvents listens to incoming events on the given file
//    and notifies the given channel ch for each one.
func (s KbdService) listenToEvents(path string, ch chan Event) error {
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

	kbdPattern := regexp.MustCompile(".*kbd.*")

	// keeps track of active listeners
	activeListeners := map[string]bool{}
	tick := time.Tick(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick:
			paths := []string{}
			for _, path := range staticPaths {
				paths = append(paths, path)
			}

			matches := util.FindFilesMatching("/dev/input/by-id", kbdPattern)
			for _, match := range matches {
				paths = append(paths, match)
			}

			for _, p := range paths {
				path, _ := filepath.EvalSymlinks(p)

				v, ok := activeListeners[path]
				if ok && v == true {
					continue
				}

				fmt.Printf("Listening to: %s\n", path)
				activeListeners[path] = true

				go func() {
					defer func() { activeListeners[path] = false }()
					s.listenToEvents(path, inputEventChannel)
				}()
			}
		}
	}
}
