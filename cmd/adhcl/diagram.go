package main

import (
	"bytes"
	"io"

	"github.com/axiomatic-design-hcl/adhcl/internal/cli"
	"github.com/axiomatic-design-hcl/adhcl/internal/diagram"
	"github.com/axiomatic-design-hcl/adhcl/internal/report"
	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

func runDiagram(request cli.Request, stdout, stderr io.Writer) (int, error) {
	res := validate.Run(validationRequest(request))
	if res.Snapshot == nil {
		if err := report.Render(stderr, res); err != nil {
			return 1, err
		}
		return 1, nil
	}
	page, err := diagram.Build(res)
	if err != nil {
		return 1, err
	}
	var buf bytes.Buffer
	if err = diagram.Render(&buf, page); err != nil {
		return 1, err
	}
	if err = deliverDiagram(stdout, request.Output, buf.Bytes()); err != nil {
		return 1, err
	}
	return 0, nil
}
