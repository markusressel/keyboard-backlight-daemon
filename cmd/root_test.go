package cmd

import (
	"io"
	"os"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/pterm/pterm"
)

func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	f()

	_ = w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestVersionCmd(t *testing.T) {
	out := captureStdout(t, func() { versionCmd.Run(versionCmd, nil) })
	if out != "1.1.0" {
		t.Fatalf("got %q", out)
	}
}

func TestSetupUi(t *testing.T) {
	t.Cleanup(func() {
		noColor, noStyle = false, false
		pterm.EnableColor()
		pterm.EnableStyling()
	})
	noColor, noStyle = true, true
	setupUi()
	if pterm.PrintColor {
		t.Fatal("color should be disabled")
	}
}

func TestPrintHeader(t *testing.T) {
	// pterm renders to its own writer; just ensure it does not panic
	printHeader()
}

var executed bool

func TestExecute_Version(t *testing.T) {
	// Execute registers flags on the global rootCmd and can only run once per process.
	if executed {
		t.Skip("Execute already ran in this process")
	}
	executed = true
	t.Setenv("HOME", t.TempDir())
	homedir.DisableCache = true
	t.Cleanup(func() { homedir.DisableCache = false })
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"keyboard-backlight-daemon", "version"}

	out := captureStdout(t, Execute)
	if out != "1.1.0" {
		t.Fatalf("got %q", out)
	}
}
