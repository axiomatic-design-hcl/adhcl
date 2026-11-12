package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestVersionUsesDevelopmentVersionByDefault(t *testing.T) {
	// Given
	binary := buildExecutable(t, "")

	// When
	output, err := exec.Command(binary, "version").CombinedOutput()

	// Then
	if err != nil {
		t.Fatalf("run version command: %v\noutput: %s", err, output)
	}
	if want := []byte("adhcl dev\n"); !bytes.Equal(output, want) {
		t.Fatalf("version output = %q, want %q", output, want)
	}
}

func TestVersionUsesLinkerInjectedVersion(t *testing.T) {
	// Given
	binary := buildExecutable(t, "-X main.buildVersion=v0.1.0")

	// When
	output, err := exec.Command(binary, "version").CombinedOutput()

	// Then
	if err != nil {
		t.Fatalf("run version command: %v\noutput: %s", err, output)
	}
	if want := []byte("adhcl v0.1.0\n"); !bytes.Equal(output, want) {
		t.Fatalf("version output = %q, want %q", output, want)
	}
}

func buildExecutable(t *testing.T, ldflags string) string {
	t.Helper()

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source file")
	}
	absoluteSourceFile, err := filepath.Abs(sourceFile)
	if err != nil {
		t.Fatalf("resolve test source path: %v", err)
	}
	repositoryRoot := filepath.Dir(filepath.Dir(filepath.Dir(absoluteSourceFile)))
	binary := filepath.Join(t.TempDir(), "adhcl")

	arguments := []string{"build", "-o", binary}
	if ldflags != "" {
		arguments = append(arguments, "-ldflags", ldflags)
	}
	arguments = append(arguments, "./cmd/adhcl")
	command := exec.Command("go", arguments...)
	command.Dir = repositoryRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build executable: %v\noutput: %s", err, output)
	}
	return binary
}
