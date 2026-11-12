package main

import (
	"io"
	"os"

	"github.com/axiomatic-design-hcl/adhcl/internal/cli"
	"github.com/axiomatic-design-hcl/adhcl/internal/report"
	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

var buildVersion = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, buildVersion, runValidation))
}

func runValidation(request cli.Request, w io.Writer) (int, error) {
	res := validate.Run(validate.Request{
		Path: request.Path, Profile: request.Profile, Candidate: request.Candidate,
		Context: request.Context, Version: request.Version,
	})
	res.Stages = append(res.Stages, validate.Stage{Name: "text_rendering", Status: "complete"})
	if err := report.Render(w, res); err != nil {
		return 1, err
	}
	return report.ExitCode(res), nil
}
