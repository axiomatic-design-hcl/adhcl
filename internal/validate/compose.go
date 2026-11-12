package validate

import (
	"fmt"
	"sort"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/owners"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

// reportHeader fills the request fields and the snapshot selection fields of the report.
func reportHeader(request Request, core coreResult) Result {
	rep := Result{
		Input:       request.Path,
		ToolVersion: request.Version,
		Contract:    string(model.LanguageCore1),
		Profile:     request.Profile,
	}
	if core.snapshot != nil {
		rep.Candidate = string(core.snapshot.Candidate)
		rep.Context = string(core.snapshot.Context)
		rep.Workload = string(core.snapshot.Workload)
	}
	return rep
}

// stagesFor lists the core stage, the owner stages, and the composition stage in order.
// Composition always completes because it combines whatever results exist, including failures.
func stagesFor(coreStatus string, ownerStages []Stage) []Stage {
	stages := []Stage{{Name: stageCore, Status: coreStatus}}
	stages = append(stages, ownerStages...)
	return append(stages, Stage{Name: stageComposition, Status: statusComplete})
}

// applySeverity sets each finding's severity for the profile. Only the severity changes.
// A code missing from the catalog is treated as an error.
func applySeverity(findings []result.Finding, profile result.Profile) {
	for i := range findings {
		severity, found := result.SeverityFor(findings[i].Code, profile)
		if !found {
			severity = result.Error
		}
		findings[i].Severity = severity
	}
}

// sortFindings orders findings by file, start line, start column, code, and subject.
// Findings without a range come first, ordered by subject and then code.
func sortFindings(findings []result.Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		return findingBefore(findings[i], findings[j])
	})
}

func findingBefore(a, b result.Finding) bool {
	if a.Range == nil || b.Range == nil {
		if (a.Range == nil) != (b.Range == nil) {
			return a.Range == nil
		}
		return subjectThenCode(a, b)
	}
	if a.Range.Filename != b.Range.Filename {
		return a.Range.Filename < b.Range.Filename
	}
	if a.Range.Start.Line != b.Range.Start.Line {
		return a.Range.Start.Line < b.Range.Start.Line
	}
	if a.Range.Start.Column != b.Range.Start.Column {
		return a.Range.Start.Column < b.Range.Start.Column
	}
	if a.Code != b.Code {
		return a.Code < b.Code
	}
	return a.Subject < b.Subject
}

func subjectThenCode(a, b result.Finding) bool {
	if a.Subject != b.Subject {
		return a.Subject < b.Subject
	}
	return a.Code < b.Code
}

// unresolvedCauses preserves structural, support, profile, and external obligations independently of severity.
func unresolvedCauses(scopes []assessment.ScopeResult, ownerResults []owners.OwnerResult, findings []result.Finding) []string {
	var causes []string
	for _, scope := range scopes {
		if scope.Classification == assessment.Unresolved {
			causes = append(causes, fmt.Sprintf("Scope %s: classification is unresolved.", scope.Matrix.Scope))
		}
	}
	for _, owner := range ownerResults {
		for _, outcome := range owner.Results {
			if outcome.Outcome == owners.OutcomeUnknown {
				causes = append(causes, fmt.Sprintf("Owner %s, attachment %s: %s", owner.Extension, outcome.Attachment, outcome.Cause))
			}
		}
	}
	for _, finding := range findings {
		if unresolvedObligation(finding.Code) {
			causes = append(causes, finding.Message)
		}
	}
	return causes
}

func unresolvedObligation(code string) bool {
	switch code {
	case result.CodeSupportGap, result.CodeIncompleteCriterion, result.CodeIncompleteModuleContract,
		result.CodeMissingLeafScenario, result.CodeIncompleteFeedbackContract,
		result.CodeUnresolvedRefinement, result.CodeUnresolvedApplicability,
		result.CodeUnresolvedInfluence, result.CodeSharedResourceReviewGap,
		result.CodeUnresolvedFeasibility, result.CodeIncomparableEvidence,
		result.CodeUnavailableExternalInput:
		return true
	}
	return false
}

// concat returns a new slice holding the findings of every list, in order.
func concat(lists ...[]result.Finding) []result.Finding {
	var total int
	for _, list := range lists {
		total += len(list)
	}
	findings := make([]result.Finding, 0, total)
	for _, list := range lists {
		findings = append(findings, list...)
	}
	return findings
}

// hasError reports whether any finding has error severity.
func hasError(findings []result.Finding) bool {
	for _, finding := range findings {
		if finding.Severity == result.Error {
			return true
		}
	}
	return false
}
