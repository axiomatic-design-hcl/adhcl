package cli

import (
	"fmt"
	"strings"
)

type invocation struct {
	command string
	help    bool
	request Request
}

func parseArguments(args []string, version string) (invocation, error) {
	if len(args) == 0 {
		return invocation{}, usageError("expected version or validate command")
	}
	if version == "" {
		version = "dev"
	}
	if args[0] == "--help" || args[0] == "-h" {
		if len(args) != 1 {
			return invocation{}, usageError("unexpected arguments after help")
		}
		return invocation{help: true}, nil
	}

	command := args[0]
	if command != "version" && command != "validate" {
		return invocation{}, usageError("unknown command %q", command)
	}
	parsed := invocation{
		command: command,
		request: Request{Profile: "source", Version: version},
	}
	commandArgs := args[1:]

	if command == "version" {
		if len(commandArgs) == 1 && isHelpFlag(commandArgs[0]) {
			parsed.help = true
			return parsed, nil
		}
		if len(commandArgs) != 0 {
			return invocation{}, usageError("version does not accept arguments")
		}
		return parsed, nil
	}

	return parseValidateArguments(parsed, commandArgs)
}

func parseValidateArguments(parsed invocation, args []string) (invocation, error) {
	seen := make(map[string]bool, 3)
	pathSeen := false
	flagsEnded := false

	for index := 0; index < len(args); index++ {
		argument := args[index]
		if !flagsEnded && !pathSeen && isHelpFlag(argument) {
			if len(args) != 1 || index != 0 {
				return invocation{}, usageError("help cannot be combined with other arguments")
			}
			parsed.help = true
			return parsed, nil
		}
		if flagsEnded || pathSeen {
			if !pathSeen && argument == "--" {
				flagsEnded = true
				continue
			}
			if pathSeen {
				return invocation{}, usageError("unexpected argument %q", argument)
			}
			parsed.request.Path = argument
			pathSeen = true
			continue
		}
		if argument == "--" {
			flagsEnded = true
			continue
		}
		if !strings.HasPrefix(argument, "-") {
			parsed.request.Path = argument
			pathSeen = true
			continue
		}

		name, value, hasValue := strings.Cut(argument, "=")
		if !isArgumentFlag(name) {
			return invocation{}, usageError("unknown flag %q", name)
		}
		if seen[name] {
			return invocation{}, usageError("flag %q was repeated", name)
		}
		seen[name] = true
		if !hasValue {
			if index+1 >= len(args) || args[index+1] == "--" || strings.HasPrefix(args[index+1], "-") {
				return invocation{}, usageError("flag %q requires a value", name)
			}
			index++
			value = args[index]
		}
		if value == "" {
			return invocation{}, usageError("flag %q requires a value", name)
		}

		switch name {
		case "--profile":
			if value != "source" && value != "architecture" {
				return invocation{}, usageError("invalid profile %q", value)
			}
			parsed.request.Profile = value
		case "--candidate":
			if !validCandidateSelector(value) {
				return invocation{}, usageError("invalid candidate selector %q", value)
			}
			parsed.request.Candidate = value
		case "--context":
			if !validContextSelector(value) {
				return invocation{}, usageError("invalid context selector %q", value)
			}
			parsed.request.Context = value
		}
	}

	if !pathSeen {
		return invocation{}, usageError("validate requires an input path")
	}
	return parsed, nil
}

func isArgumentFlag(name string) bool {
	return name == "--profile" || name == "--candidate" || name == "--context"
}

func isHelpFlag(argument string) bool {
	return argument == "--help" || argument == "-h"
}

func validCandidateSelector(selector string) bool {
	parts := strings.Split(selector, ".")
	return len(parts) == 4 && parts[0] == "decision" && validLabel(parts[1]) && parts[2] == "candidate" && validLabel(parts[3])
}

func validContextSelector(selector string) bool {
	parts := strings.Split(selector, ".")
	return len(parts) == 2 && parts[0] == "context" && validLabel(parts[1])
}

func validLabel(label string) bool {
	if len(label) == 0 || label[0] < 'a' || label[0] > 'z' {
		return false
	}
	previousUnderscore := false
	for index := 1; index < len(label); index++ {
		character := label[index]
		switch {
		case character >= 'a' && character <= 'z', character >= '0' && character <= '9':
			previousUnderscore = false
		case character == '_' && !previousUnderscore && index+1 < len(label):
			previousUnderscore = true
		default:
			return false
		}
	}
	return true
}

func usageError(format string, arguments ...any) error {
	return fmt.Errorf("adhcl: %s", fmt.Sprintf(format, arguments...))
}
