package service

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/markusressel/keyboard-backlight-daemon/internal/config"
	"github.com/markusressel/keyboard-backlight-daemon/internal/light"
	"github.com/markusressel/keyboard-backlight-daemon/internal/util"
	"github.com/oklog/run"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sync"
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
	// idleTimeout specified the time to wait before assuming that the user is idle
	idleTimeout time.Duration
	// light the light instance to control through this KbdService instance
	light light.Light
	// targetBrightness the currant target brightness
	targetBrightness float64
	// userIdle true if the user is currently idling for at least the duration of idleTimeout
	userIdle bool
	// idleUserTimer timer instance used to toggle userIdle flag
	userIdleTimer *time.Timer
}

func NewKbdService(c config.Configuration, l light.Light) *KbdService {
	return &KbdService{
		light:            l,
		idleTimeout:      c.IdleTimeout,
		userIdle:         true,
		targetBrightness: 1,
	}
}

func (s *KbdService) Run() {

	b, err := s.light.GetBrightness()
	if err != nil || b <= 0 {
		b = 1
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
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

		g.Add(func() error {
			select {
			case s := <-sig:
				fmt.Printf("Received '%s' signal, exiting...\n", s)
			case <-ctx.Done():
			}
			// returning stops all other actors, which triggers
			// controlLoop to restore the brightness before we exit
			return nil
		}, func(err error) {
			cancel()
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
			s.restoreBrightness()
			return nil
		case isActive := <-userIdleChannel:
			s.updateState(isActive)
		case <-inputEventChannel:
			s.onUserInteraction()
		}
	}
}

// restoreBrightness makes sure the light is not left off when the service exits,
// since the hardware keeps its last brightness across reboots (e.g. for the
// bootloader / disk encryption prompt before this daemon is running again).
func (s *KbdService) restoreBrightness() {
	if !s.userIdle {
		// light is already on, keep whatever the user chose
		if b, err := s.light.GetBrightness(); err == nil && b > 0 {
			return
		}
	}
	s.setBrightness(s.targetBrightness)
}

// setBrightness sets the brightness of the light and reports failures.
// A failed write must not stop the service, the next state change will retry.
func (s *KbdService) setBrightness(percentage float64) {
	if err := s.light.SetBrightness(percentage); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to set brightness to %f: %v\n", percentage, err)
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
		defer func() { s.initialized = true }()
	} else if s.userIdle == userIdle {
		return
	}

	s.userIdle = userIdle

	if userIdle {
		// update the target brightness to the
		// last brightness before detecting "idle" state
		b, err := s.light.GetBrightness()
		if err == nil && b != s.targetBrightness {
			if b <= 0 {
				// keep old targetBrightness, if new is invisible
				b = s.targetBrightness
			} else {
				fmt.Printf("Updating target brightness: %f -> %f\n", s.targetBrightness, b)
				s.targetBrightness = b
			}
		}

		s.setBrightness(0)
	} else {
		s.setBrightness(s.targetBrightness)
	}
}

// listenToEvents listens to incoming events on the given file
//
//	and notifies the given channel ch for each one.
func (s KbdService) listenToEvents(path string, ch chan Event) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open input device: %w", err)
	}
	defer f.Close()

	b := make([]byte, 24)

	for {
		_, err = io.ReadFull(f, b)
		if err != nil {
			return fmt.Errorf("failed to read from %s: %w", path, err)
		}
		event := Event{
			Sec:  binary.LittleEndian.Uint64(b[0:8]),
			Usec: binary.LittleEndian.Uint64(b[8:16]),
			Type: binary.LittleEndian.Uint16(b[16:18]),
			Code: binary.LittleEndian.Uint16(b[18:20]),
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
	var listenersMu sync.Mutex
	activeListeners := map[string]bool{}
	// last error reported per source, to avoid logging the same error every tick
	lastErrors := map[string]string{}
	reportErr := func(source string, err error) {
		listenersMu.Lock()
		defer listenersMu.Unlock()
		if lastErrors[source] == err.Error() {
			return
		}
		lastErrors[source] = err.Error()
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
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

			matches, err := util.FindFilesMatching("/dev/input/by-id", kbdPattern)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				reportErr("/dev/input/by-id", err)
			}
			for _, match := range matches {
				paths = append(paths, match)
			}

			for _, p := range paths {
				path, err := filepath.EvalSymlinks(p)
				if err != nil {
					if !errors.Is(err, fs.ErrNotExist) {
						reportErr(p, err)
					}
					continue
				}

				listenersMu.Lock()
				v, ok := activeListeners[path]
				if ok && v == true {
					listenersMu.Unlock()
					continue
				}

				fmt.Printf("Listening to: %s\n", path)
				activeListeners[path] = true
				listenersMu.Unlock()

				go func() {
					defer func() {
						listenersMu.Lock()
						activeListeners[path] = false
						listenersMu.Unlock()
					}()
					if err := s.listenToEvents(path, inputEventChannel); err != nil {
						reportErr(path, err)
					}
				}()
			}
		}
	}
}
