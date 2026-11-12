package validate

import (
	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/owners"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

// Request selects the input, profile, and invocation-local snapshot.
type Request struct {
	Path      string
	Profile   string
	Candidate string
	Context   string
	Version   string
}

// Stage records whether one operation stage completed or was blocked.
type Stage struct{ Name, Status string }

// Result retains application results independently of their presentation.
type Result struct {
	Input, ToolVersion, Contract, Profile string
	Candidate, Context, Workload          string
	Stages                                []Stage
	Findings                              []result.Finding
	Scopes                                []assessment.ScopeResult
	Candidates                            []assessment.CandidateResult
	Owners                                []owners.OwnerResult
	Unresolved                            []string
}

// Failed reports operation failure, independent of process exit conventions.
func (r Result) Failed() bool {
	for _, f := range r.Findings {
		if f.Severity == result.Error {
			return true
		}
	}
	for _, s := range r.Stages {
		if s.Status == "failed" || s.Status == "blocked" {
			return true
		}
	}
	return false
}
