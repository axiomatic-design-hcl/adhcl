package assessment

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
)

// Finite success samples do not establish feasibility; observed required failures can refute it.
const feasibilityUnresolved = "unresolved"

// CandidateResult is the feasibility and evidence of one candidate. Evidence holds one estimate
// for each matching evidence block, in authored order.
type CandidateResult struct {
	Candidate   model.ID
	Feasibility string // feasible, infeasible, unresolved
	Evidence    []EvidenceEstimate
}

// EvidenceEstimate is the two-sided Wilson interval of one evidence block with its information
// bounds. Estimates from separate blocks are never pooled.
type EvidenceEstimate struct {
	Evidence                    model.ID
	Low, High                   float64
	Best, Worst                 Information
	Trials, Failures            int64
	Confidence                  float64
	Sampling, Dependence        string
	Horizon, Version            string
	Context, Workload, Scenario model.ID
}

// exposure is the effective context, workload, and horizon of one candidate.
type exposure struct {
	context  model.ID
	workload model.ID
	horizon  string
}

// AssessCandidates assesses every candidate of every decision in the snapshot's graph. It does
// not change the snapshot.
func AssessCandidates(snapshot *selection.Snapshot) ([]CandidateResult, []result.Finding) {
	var results []CandidateResult
	var findings []result.Finding
	for i := range snapshot.Graph.Core.Decisions {
		decisionResults, decisionFindings := assessDecision(snapshot, &snapshot.Graph.Core.Decisions[i])
		results = append(results, decisionResults...)
		findings = append(findings, decisionFindings...)
	}
	return results, findings
}

// assessDecision assesses the candidates of one decision and reports whether their exposures can
// be compared.
func assessDecision(snapshot *selection.Snapshot, decision *model.Decision) ([]CandidateResult, []result.Finding) {
	core := &snapshot.Graph.Core
	var results []CandidateResult
	var findings []result.Finding
	exposures := make([]exposure, 0, len(decision.Candidates))
	for i := range decision.Candidates {
		candidate := &decision.Candidates[i]
		effective, exposureFindings := candidateExposure(snapshot, decision, candidate)
		estimates, infeasible, evidenceFindings := estimatesFor(core, decision, candidate, effective)
		feasibility := feasibilityUnresolved
		if infeasible {
			feasibility = "infeasible"
		}
		results = append(results, CandidateResult{
			Candidate:   candidate.ID,
			Feasibility: feasibility,
			Evidence:    estimates,
		})
		findings = append(findings, exposureFindings...)
		findings = append(findings, evidenceFindings...)
		if !infeasible {
			findings = append(findings, unresolvedFeasibilityFinding(candidate))
		}
		exposures = append(exposures, effective)
	}
	findings = append(findings, incomparableFindings(decision, exposures)...)
	return results, findings
}

// candidateExposure returns the effective exposure of a candidate. The selected candidate keeps
// the exposure its snapshot resolved, because the request may have supplied its context.
func candidateExposure(snapshot *selection.Snapshot, decision *model.Decision, candidate *model.Candidate) (exposure, []result.Finding) {
	if candidate.ID == snapshot.Candidate {
		return exposure{context: snapshot.Context, workload: snapshot.Workload, horizon: snapshot.Horizon}, nil
	}
	return inheritedExposure(&snapshot.Graph.Core, decision, candidate)
}

// inheritedExposure resolves the exposure of a candidate that the snapshot does not select. The
// candidate's own values win over its decision's values. A workload that disagrees with the
// candidate's context is a conflict, so the candidate keeps no workload.
func inheritedExposure(core *model.Core, decision *model.Decision, candidate *model.Candidate) (exposure, []result.Finding) {
	context := preferredID(refID(candidate.Context), refID(decision.Context))
	horizon := preferredText(candidate.Horizon, decision.Horizon)
	declared := preferredID(refID(candidate.Workload), refID(decision.Workload))
	fromContext := contextWorkload(core, context)
	if workloadsDisagree(declared, fromContext) {
		return exposure{context: context, horizon: horizon}, []result.Finding{workloadConflictFinding(candidate, declared, fromContext)}
	}
	workload := preferredID(declared, fromContext, designWorkload(core))
	return exposure{context: context, workload: workload, horizon: horizon}, nil
}

// estimatesFor keeps observed failures separate from the candidate's requested information:
// only explicitly referenced measured blocks produce estimates, and no blocks are pooled.
func estimatesFor(core *model.Core, decision *model.Decision, candidate *model.Candidate, effective exposure) ([]EvidenceEstimate, bool, []result.Finding) {
	var estimates []EvidenceEstimate
	var findings []result.Finding
	conditions := &selection.Snapshot{Context: effective.context, Workload: effective.workload}
	required, complete := candidatePredicates(core, decision, conditions)
	requested := make(map[model.ID]bool)
	if candidate.Information != nil && candidate.Information.Basis == model.InformationMeasured {
		for _, ref := range candidate.Information.Evidence {
			requested[ref.Target] = true
		}
	}
	infeasible := false
	for i := range core.Evidence {
		block := &core.Evidence[i]
		if block.Candidate.Target != candidate.ID {
			if requested[block.ID] {
				findings = append(findings, invalidEvidenceFinding(block, errors.New("referenced evidence belongs to another candidate")))
			}
			continue
		}
		estimate, err := blockEstimate(block)
		if err != nil {
			findings = append(findings, invalidEvidenceFinding(block, err))
			continue
		}
		if !matchesExposure(block, candidate, effective) || !complete ||
			!evidenceCovers(core, block, conditions, required) {
			continue
		}
		if requested[block.ID] {
			estimates = append(estimates, estimate)
		}
		if block.Failures.Sign() > 0 && !infeasible {
			infeasible = true
			span := candidate.Range
			findings = append(findings, result.Finding{
				Code:    result.CodeInfeasibleCandidate,
				Subject: string(candidate.ID),
				Message: fmt.Sprintf("Candidate %s has observed joint failures in evidence %s: at least one required criterion in %v failed.", candidate.ID, block.ID, required),
				Range:   &span,
			})
		}
	}
	return estimates, infeasible, findings
}

// candidatePredicates resolves every addressed operational predicate in the candidate's own
// conditions. Missing predicates or unresolved applicability cannot qualify a joint sample.
func candidatePredicates(core *model.Core, decision *model.Decision, conditions *selection.Snapshot) ([]model.ID, bool) {
	var required []model.ID
	complete := len(decision.Addresses) > 0
	for _, address := range decision.Addresses {
		found := false
		for i := range core.FRs {
			fr := &core.FRs[i]
			if fr.ID != address.Target {
				continue
			}
			found = true
			count := 0
			for j := range fr.Criteria {
				criterion := &fr.Criteria[j]
				switch conditions.Applies(criterion.Under) {
				case selection.Applicability("not_applicable"):
					continue
				case selection.Applicability("applies"):
					if criterion.Metric == nil || strings.TrimSpace(*criterion.Metric) == "" || criterion.Op == nil || criterion.Value == nil {
						complete = false
						continue
					}
					required = append(required, criterion.ID)
					count++
				default:
					complete = false
				}
			}
			if count == 0 {
				complete = false
			}
		}
		if !found {
			complete = false
		}
	}
	return required, complete
}

// evidenceCovers requires one applicable scenario to identify the entire joint event.
func evidenceCovers(core *model.Core, block *model.Evidence, conditions *selection.Snapshot, required []model.ID) bool {
	if block.Basis != model.EvidenceMeasured || block.Dependence != model.DependenceJoint || block.ConfidenceMethod != model.ConfidenceWilson {
		return false
	}
	for i := range core.Scenarios {
		scenario := &core.Scenarios[i]
		if scenario.ID != block.Scenario.Target || conditions.Applies(scenario.Under) != selection.Applicability("applies") {
			continue
		}
		for _, criterion := range required {
			found := false
			for _, ref := range scenario.Verifies.Items {
				if ref.Target == criterion {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return len(required) > 0
	}
	return false
}

// blockEstimate returns the Wilson interval and information bounds of one evidence block. It
// reports an error when the trial counts or the confidence violate the evidence contract.
func blockEstimate(block *model.Evidence) (EvidenceEstimate, error) {
	trials, trialsOK := wholeCount(block.Trials)
	failures, failuresOK := wholeCount(block.Failures)
	if !trialsOK || !failuresOK || block.Confidence == nil {
		return EvidenceEstimate{}, errors.New("trials and failures must be whole numbers and confidence must be a number")
	}
	confidence, _ := block.Confidence.Float64()
	low, high, err := WilsonInterval(trials, failures, confidence)
	if err != nil {
		return EvidenceEstimate{}, err
	}
	best, worst := InformationBounds(low, high)
	return EvidenceEstimate{
		Evidence: block.ID, Low: low, High: high, Best: best, Worst: worst,
		Trials: trials, Failures: failures, Confidence: confidence,
		Sampling: block.Sampling, Dependence: string(block.Dependence),
		Horizon: block.Horizon, Version: block.Version,
		Context: block.Context.Target, Workload: block.Workload.Target, Scenario: block.Scenario.Target,
	}, nil
}

// wholeCount returns the value of a count that must be a whole number within the int64 range.
func wholeCount(value *big.Float) (int64, bool) {
	if value == nil || !value.IsInt() {
		return 0, false
	}
	count, accuracy := value.Int64()
	return count, accuracy == big.Exact
}

// matchesExposure reports whether a block was observed under the candidate's exact version and
// effective exposure. A candidate without an explicit version matches no block.
func matchesExposure(block *model.Evidence, candidate *model.Candidate, effective exposure) bool {
	return candidate.Version != nil &&
		block.Version == *candidate.Version &&
		block.Context.Target == effective.context &&
		block.Workload.Target == effective.workload &&
		block.Horizon == effective.horizon
}

// incomparableFindings reports one finding when the candidates of a decision have different
// effective exposures, because their evidence then cannot be compared.
func incomparableFindings(decision *model.Decision, exposures []exposure) []result.Finding {
	if !exposuresDiffer(exposures) {
		return nil
	}
	span := decision.Range
	return []result.Finding{{
		Code:    result.CodeIncomparableEvidence,
		Subject: string(decision.ID),
		Message: fmt.Sprintf("Candidates of decision %s have different context, workload, or horizon, so their evidence cannot be compared.", decision.ID),
		Range:   &span,
	}}
}

// exposuresDiffer reports whether any exposure differs from the first.
func exposuresDiffer(exposures []exposure) bool {
	for i := range exposures {
		if exposures[i] != exposures[0] {
			return true
		}
	}
	return false
}

// invalidEvidenceFinding reports an evidence block whose counts or confidence cannot give an
// estimate.
func invalidEvidenceFinding(block *model.Evidence, err error) result.Finding {
	span := block.Range
	return result.Finding{
		Code:    result.CodeInvalidEvidence,
		Subject: string(block.ID),
		Message: fmt.Sprintf("Evidence %s cannot give an estimate: %v.", block.ID, err),
		Range:   &span,
	}
}

// unresolvedFeasibilityFinding reports a candidate without a demonstrated required failure.
// Finite success samples retain uncertainty rather than proving feasibility.
func unresolvedFeasibilityFinding(candidate *model.Candidate) result.Finding {
	span := candidate.Range
	return result.Finding{
		Code:    result.CodeUnresolvedFeasibility,
		Subject: string(candidate.ID),
		Message: fmt.Sprintf("Candidate %s has no conclusive applicable success evidence for all required criteria and constraints, so its feasibility is unresolved.", candidate.ID),
		Range:   &span,
	}
}

// workloadConflictFinding reports a candidate whose declared workload disagrees with the workload
// of its context.
func workloadConflictFinding(candidate *model.Candidate, declared, fromContext model.ID) result.Finding {
	span := candidate.Range
	return result.Finding{
		Code:    result.CodeInvalidSelection,
		Subject: string(candidate.ID),
		Message: fmt.Sprintf("Candidate %s declares workload %s, which disagrees with workload %s of its context. The candidate has no effective workload.", candidate.ID, declared, fromContext),
		Range:   &span,
	}
}

// refID returns the target of an optional reference, or the empty ID when the reference is absent.
func refID(ref *model.Ref) model.ID {
	if ref == nil {
		return ""
	}
	return ref.Target
}

// preferredID returns the first non-empty ID.
func preferredID(ids ...model.ID) model.ID {
	for _, id := range ids {
		if id != "" {
			return id
		}
	}
	return ""
}

// preferredText returns the first present text.
func preferredText(values ...*string) string {
	for _, value := range values {
		if value != nil {
			return *value
		}
	}
	return ""
}

// contextWorkload returns the workload that a context declares, or the empty ID.
func contextWorkload(core *model.Core, context model.ID) model.ID {
	for i := range core.Contexts {
		if core.Contexts[i].ID == context {
			return refID(core.Contexts[i].Workload)
		}
	}
	return ""
}

// designWorkload returns the workload that the design root declares, or the empty ID.
func designWorkload(core *model.Core) model.ID {
	if len(core.Designs) == 0 {
		return ""
	}
	return refID(core.Designs[0].Workload)
}

// workloadsDisagree reports whether two workloads are both known and different.
func workloadsDisagree(a, b model.ID) bool {
	return a != "" && b != "" && a != b
}
