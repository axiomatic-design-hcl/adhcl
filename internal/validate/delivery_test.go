package validate_test

import (
	"io"

	"github.com/axiomatic-design-hcl/adhcl/internal/report"
	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

func renderValidation(request validate.Request, w io.Writer) (int, error) {
	r := validate.Run(request)
	r.Stages = append(r.Stages, validate.Stage{Name: "text_rendering", Status: "complete"})
	if err := report.Render(w, r); err != nil {
		return 1, err
	}
	return report.ExitCode(r), nil
}
