package cli

import (
	"io"
)

const (
	exitSuccess = 0
	exitFailure = 1
	exitUsage   = 2
)

type Request struct {
	Path      string
	Profile   string
	Candidate string
	Context   string
	Version   string
}

type ValidateFunc func(Request, io.Writer) (int, error)

func Run(args []string, stdout, stderr io.Writer, version string, validate ValidateFunc) int {
	if version == "" {
		version = "dev"
	}

	invocation, err := parseArguments(args, version)
	if err != nil {
		if writeErr := writeText(stderr, err.Error()+"\n"); writeErr != nil {
			return reportDeliveryFailure(stderr, "usage error", writeErr)
		}
		return exitUsage
	}

	if invocation.help {
		if err := writeText(stdout, helpText); err != nil {
			return reportDeliveryFailure(stderr, "help", err)
		}
		return exitSuccess
	}

	if invocation.command == "version" {
		if err := writeText(stdout, "adhcl "+version+"\n"); err != nil {
			return reportDeliveryFailure(stderr, "version", err)
		}
		return exitSuccess
	}

	if validate == nil {
		_ = writeText(stderr, "adhcl: validation is unavailable\n")
		return exitFailure
	}

	code, err := validate(invocation.request, stdout)
	if err != nil {
		return reportDeliveryFailure(stderr, "validation report", err)
	}
	return code
}

const helpText = `Usage:
  adhcl version
  adhcl validate [--profile source|architecture] [--candidate REF] [--context REF] [--] PATH

Commands:
  version    Show the tool version.
  validate   Validate a core file, core directory, or explicit manifest.

Validation options:
  --profile PROFILE   Select source or architecture.
  --candidate REF     Select a candidate reference.
  --context REF       Select a context reference.
  --help, -h          Show help.

PATH is one .ad.hcl core file, a core directory, or an explicit manifest.
`

func writeText(writer io.Writer, text string) error {
	_, err := io.WriteString(writer, text)
	return err
}

func reportDeliveryFailure(stderr io.Writer, output string, cause error) int {
	_ = writeText(stderr, "adhcl: cannot deliver "+output+": "+cause.Error()+"\n")
	return exitFailure
}
