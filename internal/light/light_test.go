package light

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func newFakeLight(t *testing.T, max, current string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range map[string]string{MaxBrightness: max, Brightness: current} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestGetBrightness(t *testing.T) {
	l := NewLight(newFakeLight(t, "3\n", "1\n"))
	got, err := l.GetBrightness()
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-1.0/3.0) > 1e-9 {
		t.Fatalf("got %f, want 1/3", got)
	}
}

func TestSetBrightness(t *testing.T) {
	dir := newFakeLight(t, "4", "0")
	l := NewLight(dir)

	if err := l.SetBrightness(0.5); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, Brightness))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "2" {
		t.Fatalf("raw brightness = %q, want \"2\"", data)
	}

	got, err := l.GetBrightness()
	if err != nil || got != 0.5 {
		t.Fatalf("got (%f, %v), want 0.5", got, err)
	}
}

func TestGetBrightness_ReadError(t *testing.T) {
	dir := newFakeLight(t, "4", "0")
	l := NewLight(dir)
	if err := os.Remove(filepath.Join(dir, Brightness)); err != nil {
		t.Fatal(err)
	}
	if _, err := l.GetBrightness(); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewLight_MissingMaxBrightnessPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	NewLight(t.TempDir())
}

func TestDetectKeyboardBacklightIn(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"input3::capslock", "asus::kbd_backlight"} {
		if err := os.Mkdir(filepath.Join(dir, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got := detectKeyboardBacklightIn(dir)
	if got == nil || *got != filepath.Join(dir, "asus::kbd_backlight") {
		t.Fatalf("got %v", got)
	}
}

func TestDetectKeyboardBacklightIn_Keyboard(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "dell::keyboard"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := detectKeyboardBacklightIn(dir); got == nil {
		t.Fatal("expected match")
	}
}

func TestDetectKeyboardBacklightIn_None(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "input3::capslock"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := detectKeyboardBacklightIn(dir); got != nil {
		t.Fatalf("got %v, want nil", *got)
	}
}
