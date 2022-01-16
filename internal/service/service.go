package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/markusressel/keyboard-backlight-daemon/internal/config"
	"github.com/markusressel/keyboard-backlight-daemon/internal/light"
	"github.com/oklog/run"
	"os"
	"os/signal"
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

	userIdleChannel = make(chan bool)
)

type KbdService struct {
	idleTimeout   time.Duration
	light         light.Light
	userIdle      bool
	userIdleTimer *time.Timer
}

func NewKbdService(c config.Configuration) *KbdService {
	return &KbdService{
		light:       light.NewLight(),
		idleTimeout: c.IdleTimeout,
		userIdle:    true,
	}
}

func (s *KbdService) Run() {

	ctx, cancel := context.WithCancel(context.Background())

	var g run.Group
	{
		paths := []string{
			"/dev/input/event2",
			"/dev/input/event3",
		}

		for _, p := range paths {
			path := p
			g.Add(func() error {
				return s.listenToEvents(path, inputEventChannel)
			}, func(err error) {
				if err != nil {
					fmt.Printf("Error: %v", err)
				}
			})
		}
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
					fmt.Printf("Received %s signal, exiting...", s)
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
	if s.userIdle == userIdle {
		return
	} else {
		s.userIdle = userIdle
		fmt.Printf("UserIdle: %t\n", userIdle)
	}

	if userIdle {
		s.light.SetBrightness(100)
	} else {
		s.light.SetBrightness(0)
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
		_, _ = f.Read(b)
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
