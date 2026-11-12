package result

import "github.com/hashicorp/hcl/v2"

type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

type Finding struct {
	Code     string
	Severity Severity
	Subject  string
	Message  string
	Range    *hcl.Range
	Related  []hcl.Range
}
