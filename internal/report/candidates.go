package report

import (
	"fmt"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
)

func writeCandidates(out *lineWriter, candidates []assessment.CandidateResult) {
	if len(candidates) == 0 {
		return
	}
	out.line("Candidates")
	for _, candidate := range candidates {
		out.line(fmt.Sprintf("- %s: feasibility %s.", candidate.Candidate, candidate.Feasibility))
		for _, e := range candidate.Evidence {
			out.line(fmt.Sprintf("  Evidence %s: %d trials, %d failures; confidence %g; sampling %s; dependence %s.", e.Evidence, e.Trials, e.Failures, e.Confidence, e.Sampling, e.Dependence))
			out.line(fmt.Sprintf("  Exposure: version %s; context %s; workload %s; horizon %s; scenario %s.", e.Version, e.Context, e.Workload, e.Horizon, e.Scenario))
			out.line(fmt.Sprintf("  Joint success probability (Wilson): [%g, %g]; information bounds (bits): [%s, %s].", e.Low, e.High, informationText(e.Best), informationText(e.Worst)))
		}
	}
	out.line("")
}

func informationText(bound assessment.Information) string {
	if bound.Status == assessment.StatusFinite {
		return fmt.Sprintf("%g", bound.Value)
	}
	return bound.Status
}
