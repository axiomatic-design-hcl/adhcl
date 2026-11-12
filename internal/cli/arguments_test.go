package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestRunHelpDescribesTheRequestedCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "long root help",
			args:     []string{"--help"},
			contains: []string{"version", "validate"},
		},
		{
			name:     "short root help",
			args:     []string{"-h"},
			contains: []string{"version", "validate"},
		},
		{
			name:     "validate help",
			args:     []string{"validate", "--help"},
			contains: []string{"validate", "--profile", "--candidate", "--context"},
		},
		{
			name:     "version help",
			args:     []string{"version", "--help"},
			contains: []string{"version"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given help arguments and a callback that must not run.
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			validate := func(Request, io.Writer) (int, error) {
				panic("validation accessed for help")
			}

			// When the command handles help.
			code := Run(test.args, stdout, stderr, "test", validate)

			// Then help succeeds on stdout and describes the requested command contract.
			if code != 0 {
				t.Fatalf("Run() = %d, want 0", code)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			for _, content := range test.contains {
				if !strings.Contains(stdout.String(), content) {
					t.Errorf("help output %q does not contain %q", stdout.String(), content)
				}
			}
		})
	}
}

func TestRunRejectsInvalidUsageBeforeValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing input", args: []string{"validate"}},
		{name: "unknown command", args: []string{"inspect"}},
		{name: "removed valid alias", args: []string{"valid", "model.ad.hcl"}},
		{name: "removed strict alias", args: []string{"strict", "model.ad.hcl"}},
		{name: "version flag", args: []string{"--version"}},
		{name: "invalid source profile", args: []string{"validate", "--profile", "valid", "model.ad.hcl"}},
		{name: "invalid strict profile", args: []string{"validate", "--profile", "strict", "model.ad.hcl"}},
		{name: "unknown profile", args: []string{"validate", "--profile", "unknown", "model.ad.hcl"}},
		{name: "unknown flag", args: []string{"validate", "--unknown", "model.ad.hcl"}},
		{name: "repeated profile", args: []string{"validate", "--profile", "source", "--profile", "architecture", "model.ad.hcl"}},
		{name: "repeated candidate", args: []string{"validate", "--candidate", "decision.plan.candidate.first", "--candidate", "decision.plan.candidate.second", "model.ad.hcl"}},
		{name: "repeated context", args: []string{"validate", "--context", "context.office", "--context", "context.cloud", "model.ad.hcl"}},
		{name: "unexpected argument", args: []string{"validate", "model.ad.hcl", "extra.ad.hcl"}},
		{name: "flag after path", args: []string{"validate", "model.ad.hcl", "--profile", "source"}},
		{name: "malformed candidate selector", args: []string{"validate", "--candidate", "plan.first", "model.ad.hcl"}},
		{name: "malformed context selector", args: []string{"validate", "--context", "office", "model.ad.hcl"}},
		{name: "missing profile value", args: []string{"validate", "--profile"}},
		{name: "missing candidate value", args: []string{"validate", "--candidate"}},
		{name: "missing context value", args: []string{"validate", "--context"}},
		{name: "flag value consumed as missing", args: []string{"validate", "--profile", "--candidate", "decision.plan.candidate.first", "model.ad.hcl"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given invalid arguments and a callback that must not run.
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			validate := func(Request, io.Writer) (int, error) {
				panic("validation accessed for invalid usage")
			}

			// When the command handles the arguments.
			code := Run(test.args, stdout, stderr, "test", validate)

			// Then usage fails with code 2 and writes its error to stderr.
			if code != 2 {
				t.Fatalf("Run() = %d, want 2", code)
			}
			if stderr.Len() == 0 {
				t.Fatal("stderr is empty, want a usage error")
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
		})
	}
}
