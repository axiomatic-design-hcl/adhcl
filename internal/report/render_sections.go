package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/owners"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
	"github.com/hashicorp/hcl/v2"
)

func writeScopes(out *lineWriter, scopes []assessment.ScopeResult) {
	out.line("Scopes")
	if len(scopes) == 0 {
		out.line("- none.")
	}
	for _, scope := range scopes {
		writeScope(out, scope)
	}
	out.line("")
}

func writeScope(out *lineWriter, scope assessment.ScopeResult) {
	out.line(fmt.Sprintf("- Scope %s: %s.", scope.Matrix.Scope, scope.Classification))
	out.line("  Support: unresolved (structural classification alone is not support).")
	out.line("  Pairing: " + pairingText(scope))
	out.line("  Order: " + listOrNone(scope.Order))
	for _, cycle := range scope.Cycles {
		out.line("  Cycle: " + listOrNone(cycle))
	}
}

// pairingText shows the unique pairing or bounded ambiguity witnesses, never an exact total.
func pairingText(scope assessment.ScopeResult) string {
	if len(scope.Pairing) > 0 {
		return pairsText(scope.Pairing)
	}
	if len(scope.PairingWitnesses) > 1 {
		return fmt.Sprintf("ambiguous, with %d complete pairing witnesses (at least two pairings)", len(scope.PairingWitnesses))
	}
	return "none"
}

// pairsText lists each requirement with its design parameter, sorted by requirement.
func pairsText(pairing map[model.ID]model.ID) string {
	requirements := make([]model.ID, 0, len(pairing))
	for requirement := range pairing {
		requirements = append(requirements, requirement)
	}
	sort.Slice(requirements, func(i, j int) bool {
		return requirements[i] < requirements[j]
	})
	parts := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		parts = append(parts, fmt.Sprintf("%s with %s", requirement, pairing[requirement]))
	}
	return strings.Join(parts, ", ")
}

func writeOwners(out *lineWriter, results []owners.OwnerResult) {
	out.line("Owners")
	if len(results) == 0 {
		out.line("- none.")
	}
	for _, owner := range results {
		writeOwner(out, owner)
	}
	out.line("")
}

func writeOwner(out *lineWriter, owner owners.OwnerResult) {
	out.line(fmt.Sprintf("- %s: %s.", owner.Extension, owner.Status))
	out.line(fmt.Sprintf("  Contract: %s; implementation: %s; source: %s", owner.Contract, owner.Implementation, sourceRangeText(owner.Range)))
	out.line(fmt.Sprintf("  Core revision: %s; snapshot: %s; invocation: %s", owner.CoreRevision, owner.Snapshot, owner.Invocation))
	for _, source := range owner.CoreSources {
		out.line(fmt.Sprintf("  Core source: %s; revision: %s", source.Path, source.Revision))
	}
	for _, attachment := range owner.Results {
		out.line("  " + attachmentText(attachment))
	}
}

// attachmentText names the attachment, its subject, and its outcome. It adds
// the cause when the outcome has one.
func attachmentText(attachment owners.AttachmentResult) string {
	text := fmt.Sprintf("- %s on %s: %s.", attachment.Attachment, attachment.Subject, attachment.Outcome)
	if attachment.Cause != "" {
		text += " Cause: " + attachment.Cause
	}
	if attachment.Revision != "" {
		text += fmt.Sprintf(" Input: %s; revision: %s", sourceRangeText(attachment.Range), attachment.Revision)
	}
	return text
}

func writeSummary(out *lineWriter, report validate.Result) {
	errorCount, warningCount := countSeverities(report.Findings)
	out.line("Summary")
	out.line(fmt.Sprintf("Errors: %d", errorCount))
	out.line(fmt.Sprintf("Warnings: %d", warningCount))
	out.line("Stages:")
	writeStages(out, report.Stages)
	out.line("Unresolved causes:")
	writeUnresolved(out, report.Unresolved)
	out.line("")
}

func countSeverities(findings []result.Finding) (int, int) {
	errorCount, warningCount := 0, 0
	for _, finding := range findings {
		switch finding.Severity {
		case result.Error:
			errorCount++
		case result.Warning:
			warningCount++
		}
	}
	return errorCount, warningCount
}

func writeStages(out *lineWriter, stages []validate.Stage) {
	if len(stages) == 0 {
		out.line("- none.")
	}
	for _, stage := range stages {
		out.line(fmt.Sprintf("- %s: %s.", stage.Name, stage.Status))
	}
}

func writeUnresolved(out *lineWriter, causes []string) {
	if len(causes) == 0 {
		out.line("- none.")
	}
	for _, cause := range causes {
		out.line("- " + cause)
	}
}

func listOrNone(ids []model.ID) string {
	if len(ids) == 0 {
		return "none"
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = string(id)
	}
	return strings.Join(parts, ", ")
}

func sourceRangeText(r hcl.Range) string {
	return fmt.Sprintf("%s:%d:%d-%d:%d", r.Filename, r.Start.Line, r.Start.Column, r.End.Line, r.End.Column)
}
