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
	Output    string
	Path      string
	Profile   string
	Candidate string
	Context   string
	Version   string
}

type ValidateFunc func(Request, io.Writer) (int, error)
type DiagramFunc func(Request, io.Writer, io.Writer) (int, error)

func Run(args []string, stdout, stderr io.Writer, version string, validate ValidateFunc, diagram DiagramFunc) int {
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
	if invocation.command == "diagram" {
		if diagram == nil {
			_ = writeText(stderr, "adhcl: diagram is unavailable\n")
			return exitFailure
		}
		code, err := diagram(invocation.request, stdout, stderr)
		if err != nil {
			return reportDeliveryFailure(stderr, "diagram", err)
		}
		return code
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
  adhcl diagram [--profile source|architecture] [--candidate REF] [--context REF] [-o FILE|--output FILE] [--] PATH

Commands:
  version    Show the tool version.
  validate   Validate a core file, core directory, or explicit manifest.
  diagram    Render an assessed design as HTML.

Validation options:
  --profile PROFILE   Select source or architecture.
  --candidate REF     Select a candidate reference.
  --context REF       Select a context reference.
  --help, -h          Show help.

Diagram options:
  -o, --output FILE   Write the diagram to a file.

The diagram command also accepts the validation options.

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
