package util

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadIntFromFile(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name    string
		content string
		want    int
		wantErr bool
	}{
		{"plain", "42", 42, false},
		{"trailing newline", "7\n", 7, false},
		{"empty", "", 0, true},
		{"not a number", "abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := filepath.Join(dir, "f")
			writeFile(t, p, tt.content)
			got, err := ReadIntFromFile(p)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestReadIntFromFile_Missing(t *testing.T) {
	got, err := ReadIntFromFile(filepath.Join(t.TempDir(), "nope"))
	if err == nil || got != -1 {
		t.Fatalf("got (%d, %v), want (-1, error)", got, err)
	}
}

func TestWriteIntToFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	// pre-create: WriteIntToFile passes perm 644 (decimal) for new files
	writeFile(t, target, "")

	if err := WriteIntToFile(123, target); err != nil {
		t.Fatal(err)
	}
	got, err := ReadIntFromFile(target)
	if err != nil || got != 123 {
		t.Fatalf("got (%d, %v), want 123", got, err)
	}
}

func TestWriteIntToFile_FollowsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	writeFile(t, target, "")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if err := WriteIntToFile(5, link); err != nil {
		t.Fatal(err)
	}
	got, err := ReadIntFromFile(target)
	if err != nil || got != 5 {
		t.Fatalf("got (%d, %v), want 5", got, err)
	}
}

func TestFindFilesMatching(t *testing.T) {
	dir := t.TempDir()
	real1 := filepath.Join(dir, "real-kbd-1")
	other := filepath.Join(dir, "mouse")
	writeFile(t, real1, "")
	writeFile(t, other, "")

	byID := filepath.Join(dir, "by-id")
	if err := os.Mkdir(byID, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real1, filepath.Join(byID, "usb-event-kbd")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(byID, "usb-mouse"), "")

	got, err := FindFilesMatching(dir, regexp.MustCompile(".*kbd.*"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)

	// resolved real path is returned for both the file and the symlink
	resolved, err := filepath.EvalSymlinks(real1)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{resolved, resolved}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestFindFilesMatching_NoMatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a"), "")
	got, err := FindFilesMatching(dir, regexp.MustCompile("zzz"))
	if err != nil || len(got) != 0 {
		t.Fatalf("got (%v, %v), want empty", got, err)
	}
}

func TestFindFilesMatching_MissingDir(t *testing.T) {
	got, err := FindFilesMatching(filepath.Join(t.TempDir(), "nope"), regexp.MustCompile(".*"))
	if !errors.Is(err, fs.ErrNotExist) || len(got) != 0 {
		t.Fatalf("got (%v, %v), want not-exist error", got, err)
	}
}

func TestFindFilesMatching_BrokenSymlinkIsReportedAndSkipped(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good-kbd")
	writeFile(t, good, "")
	if err := os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "broken-kbd")); err != nil {
		t.Fatal(err)
	}

	got, err := FindFilesMatching(dir, regexp.MustCompile(".*kbd.*"))
	if err == nil {
		t.Fatal("expected error for broken symlink")
	}
	if len(got) != 1 {
		t.Fatalf("got %v, want only the valid file", got)
	}
}
