// Package validate runs the application validation pipeline without presentation.
package validate

import (
	"github.com/axiomatic-design-hcl/adhcl/internal/input"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

const (
	statusComplete = "complete"
)

// Run validates the request and returns all composed results without rendering.
func Run(request Request) Result {
	selected := input.Select(request.Path)
	core := assessCore(selected, request)
	ownerResults, ownerFindings, ownerStages := assessOwners(selected, core)
	attachProvenance(selected, core, request, ownerResults)

	findings := concat(selected.Findings, core.findings, ownerFindings)
	applySeverity(findings, result.Profile(request.Profile))
	sortFindings(findings)

	rep := reportHeader(request, core)
	rep.Stages = stagesFor(core.status, ownerStages)
	rep.Findings = findings
	rep.Scopes = core.scopes
	rep.Owners = ownerResults
	rep.Candidates = core.candidates
	rep.Unresolved = unresolvedCauses(core.scopes, ownerResults, findings)
	return rep
}
