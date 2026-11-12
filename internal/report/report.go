// Package report presents application validation results as plain text.
package report

import "github.com/axiomatic-design-hcl/adhcl/internal/validate"

// ExitCode translates application failure to the command's exit convention.
func ExitCode(result validate.Result) int {
	if result.Failed() {
		return 1
	}
	return 0
}
