package service

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/markusressel/keyboard-backlight-daemon/internal/config"
)

type fakeLight struct {
	mu         sync.Mutex
	brightness float64
	getErr     error
	sets       []float64
}

func (f *fakeLight) SetBrightness(p float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sets = append(f.sets, p)
	f.brightness = p
	return nil
}

func (f *fakeLight) GetBrightness() (float64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.brightness, f.getErr
}

func (f *fakeLight) lastSet() (float64, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.sets) == 0 {
		return 0, false
	}
	return f.sets[len(f.sets)-1], true
}

func TestNewKbdService(t *testing.T) {
	l := &fakeLight{}
	s := NewKbdService(config.Configuration{IdleTimeout: 3 * time.Second}, l)
	if s.idleTimeout != 3*time.Second || !s.userIdle || s.targetBrightness != 1 || s.initialized {
		t.Fatalf("unexpected state: %+v", s)
	}
}

func TestUpdateState_IdleStoresBrightnessAndTurnsOff(t *testing.T) {
	l := &fakeLight{brightness: 0.5}
	s := NewKbdService(config.Configuration{}, l)

	s.updateState(true)

	if s.targetBrightness != 0.5 {
		t.Fatalf("targetBrightness = %f, want 0.5", s.targetBrightness)
	}
	if got, _ := l.lastSet(); got != 0 {
		t.Fatalf("last set = %f, want 0", got)
	}
	if !s.initialized {
		t.Fatal("expected initialized")
	}
}

func TestUpdateState_ActiveRestoresTarget(t *testing.T) {
	l := &fakeLight{brightness: 0.5}
	s := NewKbdService(config.Configuration{}, l)
	s.updateState(true)
	s.updateState(false)

	if got, _ := l.lastSet(); got != 0.5 {
		t.Fatalf("last set = %f, want 0.5", got)
	}
	if s.userIdle {
		t.Fatal("userIdle should be false")
	}
}

func TestUpdateState_NoChangeIsIgnored(t *testing.T) {
	l := &fakeLight{brightness: 0.5}
	s := NewKbdService(config.Configuration{}, l)
	s.updateState(true)
	n := len(l.sets)

	s.updateState(true)

	if len(l.sets) != n {
		t.Fatalf("expected no additional writes, got %d -> %d", n, len(l.sets))
	}
}

func TestUpdateState_ZeroBrightnessKeepsOldTarget(t *testing.T) {
	l := &fakeLight{brightness: 0}
	s := NewKbdService(config.Configuration{}, l)
	s.targetBrightness = 0.75

	s.updateState(true)

	if s.targetBrightness != 0.75 {
		t.Fatalf("targetBrightness = %f, want 0.75", s.targetBrightness)
	}
}

func TestUpdateState_ReadErrorKeepsTarget(t *testing.T) {
	l := &fakeLight{brightness: 0.2, getErr: os.ErrNotExist}
	s := NewKbdService(config.Configuration{}, l)
	s.targetBrightness = 0.9

	s.updateState(true)

	if s.targetBrightness != 0.9 {
		t.Fatalf("targetBrightness = %f, want 0.9", s.targetBrightness)
	}
}

func inputEventBytes(sec, usec uint64, typ, code uint16, value int32) []byte {
	b := make([]byte, 24)
	binary.LittleEndian.PutUint64(b[0:8], sec)
	binary.LittleEndian.PutUint64(b[8:16], usec)
	binary.LittleEndian.PutUint16(b[16:18], typ)
	binary.LittleEndian.PutUint16(b[18:20], code)
	binary.LittleEndian.PutUint32(b[20:24], uint32(value))
	return b
}

func TestListenToEvents(t *testing.T) {
	p := filepath.Join(t.TempDir(), "events")
	data := append(inputEventBytes(1, 2, 3, 4, 5), inputEventBytes(10, 20, 1, 30, -1)...)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}

	ch := make(chan Event, 2)
	s := KbdService{}
	err := s.listenToEvents(p, ch) // returns at EOF
	if err == nil {
		t.Fatal("expected EOF error")
	}

	got := map[Event]bool{}
	for i := 0; i < 2; i++ {
		select {
		case e := <-ch:
			got[e] = true
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event")
		}
	}
	for _, want := range []Event{{1, 2, 3, 4}, {10, 20, 1, 30}} {
		if !got[want] {
			t.Fatalf("missing event %+v in %v", want, got)
		}
	}
}

func TestListenToEvents_MissingFile(t *testing.T) {
	s := KbdService{}
	err := s.listenToEvents(filepath.Join(t.TempDir(), "nope"), make(chan Event, 1))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v, want not-exist error", err)
	}
}

func TestListenToEvents_TruncatedEvent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "events")
	if err := os.WriteFile(p, inputEventBytes(1, 2, 3, 4, 5)[:10], 0o644); err != nil {
		t.Fatal(err)
	}
	s := KbdService{}
	err := s.listenToEvents(p, make(chan Event, 1))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v, want unexpected EOF", err)
	}
}

type failingLight struct{ fakeLight }

func (f *failingLight) SetBrightness(float64) error { return errors.New("boom") }

func TestUpdateState_SetBrightnessErrorDoesNotPanic(t *testing.T) {
	l := &failingLight{fakeLight{brightness: 0.5}}
	s := NewKbdService(config.Configuration{}, l)
	s.updateState(true)
	s.updateState(false)
	s.restoreBrightness()
}

func TestWatchInputDevices_StopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := &KbdService{}
	if err := s.watchInputDevices(ctx); err != nil {
		t.Fatalf("got %v, want nil", err)
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestControlLoop(t *testing.T) {
	l := &fakeLight{brightness: 0.6}
	s := NewKbdService(config.Configuration{IdleTimeout: 50 * time.Millisecond}, l)
	s.targetBrightness = 0.6

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.controlLoop(ctx) }()

	// idle timer fires -> light off
	waitFor(t, "light off", func() bool {
		v, ok := l.lastSet()
		return ok && v == 0
	})

	// user activity -> light restored
	inputEventChannel <- Event{}
	waitFor(t, "light restored", func() bool {
		v, _ := l.lastSet()
		return v == 0.6
	})

	// exit restores brightness
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("controlLoop returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("controlLoop did not exit")
	}
	if v, _ := l.lastSet(); v != 0.6 {
		t.Fatalf("final brightness = %f, want 0.6", v)
	}
}

func TestRestoreBrightness_IdleTurnsLightBackOn(t *testing.T) {
	l := &fakeLight{brightness: 0}
	s := NewKbdService(config.Configuration{}, l)
	s.targetBrightness = 0.7
	s.userIdle = true

	s.restoreBrightness()

	if got, _ := l.lastSet(); got != 0.7 {
		t.Fatalf("last set = %f, want 0.7", got)
	}
}

func TestRestoreBrightness_ActiveKeepsUserBrightness(t *testing.T) {
	l := &fakeLight{brightness: 0.3}
	s := NewKbdService(config.Configuration{}, l)
	s.targetBrightness = 0.7
	s.userIdle = false

	s.restoreBrightness()

	if len(l.sets) != 0 {
		t.Fatalf("expected no writes, got %v", l.sets)
	}
}

func TestRestoreBrightness_ActiveButLightOffRestoresTarget(t *testing.T) {
	l := &fakeLight{brightness: 0}
	s := NewKbdService(config.Configuration{}, l)
	s.targetBrightness = 0.7
	s.userIdle = false

	s.restoreBrightness()

	if got, _ := l.lastSet(); got != 0.7 {
		t.Fatalf("last set = %f, want 0.7", got)
	}
}
