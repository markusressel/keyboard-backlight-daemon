package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

func reset(t *testing.T) {
	t.Helper()
	viper.Reset()
	CurrentConfig = Configuration{}
	t.Cleanup(func() {
		viper.Reset()
		CurrentConfig = Configuration{}
	})
}

func TestReadConfigFile_FromFlagPath(t *testing.T) {
	reset(t)
	p := filepath.Join(t.TempDir(), "cfg.yaml")
	content := `backlightPath: "/sys/class/leds/x::kbd_backlight"
inputEventDevices:
  - "/dev/input/a"
  - "/dev/input/b"
idleTimeout: 15s
`
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	InitConfig(p)
	ReadConfigFile()

	if CurrentConfig.BacklightPath == nil || *CurrentConfig.BacklightPath != "/sys/class/leds/x::kbd_backlight" {
		t.Fatalf("BacklightPath = %v", CurrentConfig.BacklightPath)
	}
	if len(CurrentConfig.InputEventDevices) != 2 || CurrentConfig.InputEventDevices[1] != "/dev/input/b" {
		t.Fatalf("InputEventDevices = %v", CurrentConfig.InputEventDevices)
	}
	if CurrentConfig.IdleTimeout != 15*time.Second {
		t.Fatalf("IdleTimeout = %v", CurrentConfig.IdleTimeout)
	}
}

func TestReadConfigFile_DefaultIdleTimeout(t *testing.T) {
	reset(t)
	p := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(p, []byte("inputEventDevices: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	InitConfig(p)
	ReadConfigFile()

	if CurrentConfig.IdleTimeout != 6*time.Second {
		t.Fatalf("IdleTimeout = %v, want 6s", CurrentConfig.IdleTimeout)
	}
	if CurrentConfig.BacklightPath != nil {
		t.Fatalf("BacklightPath = %v, want nil", *CurrentConfig.BacklightPath)
	}
}

func TestReadConfigFile_MissingFile(t *testing.T) {
	reset(t)
	InitConfig(filepath.Join(t.TempDir(), "missing.yaml"))
	ReadConfigFile() // must not panic or exit

	if CurrentConfig.BacklightPath != nil || len(CurrentConfig.InputEventDevices) != 0 {
		t.Fatalf("unexpected config: %+v", CurrentConfig)
	}
}

func TestInitConfig_SearchPaths(t *testing.T) {
	reset(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	homedir.DisableCache = true
	t.Cleanup(func() { homedir.DisableCache = false })
	if err := os.Mkdir(filepath.Join(home, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(home, ".config", "keyboard-backlight-daemon.yaml")
	if err := os.WriteFile(p, []byte("idleTimeout: 2s\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	InitConfig("")
	ReadConfigFile()

	if CurrentConfig.IdleTimeout != 2*time.Second {
		t.Fatalf("IdleTimeout = %v, want 2s", CurrentConfig.IdleTimeout)
	}
}
