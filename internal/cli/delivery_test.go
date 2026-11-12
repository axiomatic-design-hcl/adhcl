package cli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestVersionWritesExactDevelopmentOutputWithoutValidation(t *testing.T) {
	// Given
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validate := func(Request, io.Writer) (int, error) {
		panic("version must not invoke validation")
	}

	// When
	code := Run([]string{"version"}, &stdout, &stderr, "", validate)

	// Then
	if code != 0 {
		t.Fatalf("Run() = %d, want 0", code)
	}
	if got, want := stdout.String(), "adhcl dev\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestVersionWritesExactInjectedOutputWithoutValidation(t *testing.T) {
	// Given
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validate := func(Request, io.Writer) (int, error) {
		panic("version must not invoke validation")
	}

	// When
	code := Run([]string{"version"}, &stdout, &stderr, "v0.1.0", validate)

	// Then
	if code != 0 {
		t.Fatalf("Run() = %d, want 0", code)
	}
	if got, want := stdout.String(), "adhcl v0.1.0\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestVersionOutputFailureReportsDeliveryFailure(t *testing.T) {
	// Given
	var stderr bytes.Buffer
	validate := func(Request, io.Writer) (int, error) {
		panic("version must not invoke validation")
	}

	stdoutFailure := errors.New("stdout unavailable")
	code := Run([]string{"version"}, failingWriter{err: stdoutFailure}, &stderr, "v0.1.0", validate)

	// Then
	if code != 1 {
		t.Fatalf("Run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), stdoutFailure.Error()) {
		t.Fatalf("stderr = %q, want the output failure cause %q", stderr.String(), stdoutFailure)
	}
}

func TestHelpOutputFailureReportsDeliveryFailureWithoutValidation(t *testing.T) {
	// Given
	var stderr bytes.Buffer
	validate := func(Request, io.Writer) (int, error) {
		panic("help must not invoke validation")
	}

	stdoutFailure := errors.New("stdout unavailable")
	code := Run([]string{"--help"}, failingWriter{err: stdoutFailure}, &stderr, "dev", validate)

	// Then
	if code != 1 {
		t.Fatalf("Run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), stdoutFailure.Error()) {
		t.Fatalf("stderr = %q, want the output failure cause %q", stderr.String(), stdoutFailure)
	}
}

func TestValidationOutputFailureReportsDeliveryFailure(t *testing.T) {
	// Given
	var stderr bytes.Buffer
	deliveryErr := errors.New("report output unavailable")
	validate := func(_ Request, output io.Writer) (int, error) {
		_, err := io.WriteString(output, "validation report\n")
		return 1, err
	}

	// When
	code := Run([]string{"validate", "design.ad.hcl"}, failingWriter{err: deliveryErr}, &stderr, "dev", validate)

	// Then
	if code != 1 {
		t.Fatalf("Run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), deliveryErr.Error()) {
		t.Fatalf("stderr = %q, want the output failure cause %q", stderr.String(), deliveryErr)
	}
}

type failingWriter struct {
	err error
}

func (writer failingWriter) Write([]byte) (int, error) {
	return 0, writer.err
}
