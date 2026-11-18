package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestDeliverDiagram(t *testing.T) {
	var out bytes.Buffer
	if err := deliverDiagram(&out, "", []byte("page")); err != nil || out.String() != "page" {
		t.Fatalf("stdout: %q, %v", out.String(), err)
	}
	if err := deliverDiagram(shortWriter{}, "", []byte("page")); err != io.ErrShortWrite {
		t.Fatalf("short write: %v", err)
	}
	path := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := deliverDiagram(&out, path, []byte("new")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "new" {
		t.Fatalf("file: %q, %v", got, err)
	}
	if err := deliverDiagram(&out, filepath.Join(path, "missing", "page.html"), []byte("bad")); err == nil {
		t.Fatal("missing directory accepted")
	}
	got, err = os.ReadFile(path)
	if err != nil || string(got) != "new" {
		t.Fatalf("existing file changed: %q, %v", got, err)
	}
}

func TestDeliverDiagramFileModes(t *testing.T) {
	for _, tc := range []struct {
		name     string
		existing bool
		mode     os.FileMode
	}{
		{"new file", false, 0644},
		{"existing readable file", true, 0644},
		{"existing restricted file", true, 0640},
		{"existing executable file", true, 0751},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "page.html")
			if tc.existing {
				if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(path, tc.mode); err != nil {
					t.Fatal(err)
				}
			}
			if err := deliverDiagram(io.Discard, path, []byte("new")); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != tc.mode {
				t.Fatalf("mode = %04o, want %04o", got, tc.mode)
			}
		})
	}
}

func TestDeliverDiagramErrorNamesOnlyDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "destination")
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	err := deliverDiagram(io.Discard, path, []byte("page"))
	if err == nil || !strings.Contains(err.Error(), path) || strings.Contains(err.Error(), ".adhcl-diagram-") {
		t.Fatalf("delivery error = %v, want destination without temporary path", err)
	}
}
