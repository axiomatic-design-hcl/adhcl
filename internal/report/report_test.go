package report

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/owners"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
	"github.com/hashicorp/hcl/v2"
)

const (
	reportTestDesignFile   = "design.ad.hcl"
	reportTestBindingFile  = "bindings.ad.ext.hcl"
	reportTestBindingCause = "No caller-supplied observations are available."
)

var (
	reportTestColonDigits  = regexp.MustCompile(`:\d+`)
	reportTestErrorCount   = regexp.MustCompile(`(?i)\berrors?\b.*\b2\b|\b2\b.*\berrors?\b`)
	reportTestWarningCount = regexp.MustCompile(`(?i)\bwarnings?\b.*\b3\b|\b3\b.*\bwarnings?\b`)
)

// reportTestBaseReport returns a report with the required header values and completed stages.
func reportTestBaseReport(profile string) validate.Result {
	return validate.Result{
		Input:       reportTestDesignFile,
		ToolVersion: "v0.1.0",
		Contract:    "requirements_hcl/core/1",
		Profile:     profile,
		Stages: []validate.Stage{
			{Name: "core_assessment", Status: "complete"},
			{Name: "composition", Status: "complete"},
			{Name: "text_rendering", Status: "complete"},
		},
	}
}

func reportTestRender(t *testing.T, report validate.Result) string {
	t.Helper()
	var output bytes.Buffer
	if err := Render(&output, report); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	return output.String()
}

// reportTestFindLine returns the first output line that contains every part, ignoring case.
func reportTestFindLine(t *testing.T, output string, parts ...string) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if reportTestLineHasAll(line, parts) {
			return line
		}
	}
	t.Fatalf("no output line contains all of %q; output:\n%s", parts, output)
	return ""
}

func reportTestLineHasAll(line string, parts []string) bool {
	lower := strings.ToLower(line)
	for _, part := range parts {
		if !strings.Contains(lower, strings.ToLower(part)) {
			return false
		}
	}
	return true
}

func reportTestRequireLine(t *testing.T, output string, parts ...string) {
	t.Helper()
	reportTestFindLine(t, output, parts...)
}

func reportTestRequireText(t *testing.T, output, text string) {
	t.Helper()
	if !strings.Contains(output, text) {
		t.Fatalf("output lacks %q; output:\n%s", text, output)
	}
}

func reportTestPosition(t *testing.T, output, text string) int {
	t.Helper()
	position := strings.Index(output, text)
	if position < 0 {
		t.Fatalf("output lacks %q; output:\n%s", text, output)
	}
	return position
}

func reportTestRequireMatch(t *testing.T, output string, pattern *regexp.Regexp, what string) {
	t.Helper()
	if !pattern.MatchString(output) {
		t.Fatalf("output lacks %s; output:\n%s", what, output)
	}
}

// reportTestFailingWriter rejects every write with its error.
type reportTestFailingWriter struct {
	err error
}

func (w reportTestFailingWriter) Write([]byte) (int, error) {
	return 0, w.err
}

// reportTestLimitedWriter accepts output up to limit bytes, then rejects every later write.
type reportTestLimitedWriter struct {
	limit              int
	err                error
	accepted           bytes.Buffer
	failed             bool
	writesAfterFailure int
}

func (w *reportTestLimitedWriter) Write(p []byte) (int, error) {
	if w.failed {
		w.writesAfterFailure++
		return 0, w.err
	}
	if w.accepted.Len()+len(p) > w.limit {
		w.failed = true
		return 0, w.err
	}
	return w.accepted.Write(p)
}

func TestRenderHeaderNamesRequestedValues(t *testing.T) {
	// Given a report with an explicit candidate, context, and workload.
	report := reportTestBaseReport("architecture")
	report.Candidate = "decision.receipt.candidate.fast"
	report.Context = "context.review"
	report.Workload = "workload.peak"

	// When the report renders.
	output := reportTestRender(t, report)

	// Then the header names the input, tool version, contract, profile, and each requested value.
	for _, want := range []string{
		reportTestDesignFile,
		"v0.1.0",
		"requirements_hcl/core/1",
		"architecture",
		"decision.receipt.candidate.fast",
		"context.review",
		"workload.peak",
	} {
		reportTestRequireText(t, output, want)
	}
}

func TestRenderHeaderUsesDefaultCandidateAndNoneForAbsentContextAndWorkload(t *testing.T) {
	// Given a report without candidate, context, or workload.
	report := reportTestBaseReport("source")

	// When the report renders.
	output := reportTestRender(t, report)

	// Then the candidate line shows default and the context and workload lines show none.
	reportTestRequireLine(t, output, "candidate", "default")
	reportTestRequireLine(t, output, "context", "none")
	reportTestRequireLine(t, output, "workload", "none")
}

func TestRenderRangeFindingShowsSeverityCodeLocationAndMessage(t *testing.T) {
	// Given an error finding with a source range at design.ad.hcl line 12, column 3.
	report := reportTestBaseReport("source")
	report.Findings = []result.Finding{{
		Code:     "invalid_reference",
		Severity: result.Error,
		Subject:  "dp.first_control",
		Message:  "The traversal targets a decision.",
		Range: &hcl.Range{
			Filename: reportTestDesignFile,
			Start:    hcl.Pos{Line: 12, Column: 3, Byte: 210},
			End:      hcl.Pos{Line: 12, Column: 20, Byte: 227},
		},
	}}

	// When the report renders.
	output := reportTestRender(t, report)

	// Then the finding line shows its severity, code, message, and file:line:column.
	reportTestRequireLine(t, output, "error", "invalid_reference", "The traversal targets a decision.", "design.ad.hcl:12:3")
}

func TestRenderPathFindingWithoutRangeInventsNoLine(t *testing.T) {
	// Given an input failure that names a path and has no source range.
	report := reportTestBaseReport("source")
	report.Findings = []result.Finding{{
		Code:     "input_failure",
		Severity: result.Error,
		Subject:  reportTestBindingFile,
		Message:  "The file cannot be read.",
	}}

	// When the report renders.
	output := reportTestRender(t, report)

	// Then the finding line names the path and contains no line or column number.
	line := reportTestFindLine(t, output, "input_failure", reportTestBindingFile)
	if reportTestColonDigits.MatchString(line) {
		t.Fatalf("path finding line = %q, want no invented line or column", line)
	}
}

func TestRenderRelatedRangesAppearAfterTheFinding(t *testing.T) {
	// Given a duplicate identity finding at design.ad.hcl line 12 that relates to a second declaration at line 40.
	report := reportTestBaseReport("source")
	report.Findings = []result.Finding{{
		Code:     "duplicate_identity",
		Severity: result.Error,
		Subject:  "dp.first_control",
		Message:  "The label is declared twice.",
		Range: &hcl.Range{
			Filename: reportTestDesignFile,
			Start:    hcl.Pos{Line: 12, Column: 3, Byte: 210},
			End:      hcl.Pos{Line: 12, Column: 20, Byte: 227},
		},
		Related: []hcl.Range{{
			Filename: reportTestDesignFile,
			Start:    hcl.Pos{Line: 40, Column: 5, Byte: 900},
			End:      hcl.Pos{Line: 40, Column: 18, Byte: 913},
		}},
	}}

	// When the report renders.
	output := reportTestRender(t, report)

	// Then the related location appears in the output after the finding message.
	message := reportTestPosition(t, output, "The label is declared twice.")
	related := reportTestPosition(t, output, "design.ad.hcl:40:5")
	if related <= message {
		t.Fatalf("related range at %d, finding message at %d; want related range after the message", related, message)
	}
}

func TestRenderKeepsFindingsInGivenOrder(t *testing.T) {
	// Given two findings whose given order differs from their subject and code order.
	report := reportTestBaseReport("source")
	report.Findings = []result.Finding{
		{Code: "invalid_syntax", Severity: result.Error, Subject: "dp.zeta", Message: "Zeta finding is first in the given list."},
		{Code: "invalid_reference", Severity: result.Error, Subject: "dp.alpha", Message: "Alpha finding is second in the given list."},
	}

	// When the report renders.
	output := reportTestRender(t, report)

	// Then the first given finding appears before the second.
	first := reportTestPosition(t, output, "Zeta finding is first in the given list.")
	second := reportTestPosition(t, output, "Alpha finding is second in the given list.")
	if first >= second {
		t.Fatalf("first finding at %d, second at %d; want given order", first, second)
	}
}

func TestRenderSummaryCountsErrorsAndWarnings(t *testing.T) {
	// Given two error findings and three warning findings without source ranges.
	report := reportTestBaseReport("source")
	report.Findings = []result.Finding{
		{Code: "invalid_reference", Severity: result.Error, Subject: "dp.alpha", Message: "Alpha reference is wrong."},
		{Code: "invalid_syntax", Severity: result.Error, Subject: "dp.beta", Message: "Beta syntax is wrong."},
		{Code: "support_gap", Severity: result.Warning, Subject: "dp.alpha", Message: "Alpha claim lacks support."},
		{Code: "unavailable_external_input", Severity: result.Warning, Subject: "module.total", Message: "Total behavior is unknown."},
		{Code: "unresolved_feasibility", Severity: result.Warning, Subject: "decision.fast", Message: "Fast candidate lacks evidence."},
	}

	// When the report renders.
	output := reportTestRender(t, report)

	// Then the summary shows two errors and three warnings.
	reportTestRequireMatch(t, output, reportTestErrorCount, "an error count of 2")
	reportTestRequireMatch(t, output, reportTestWarningCount, "a warning count of 3")
}

func TestRenderSummaryListsEachStageStatus(t *testing.T) {
	// Given four stages with four different statuses.
	report := reportTestBaseReport("source")
	report.Stages = []validate.Stage{
		{Name: "core_assessment", Status: "complete"},
		{Name: "owner:terraform_binding", Status: "failed"},
		{Name: "composition", Status: "blocked"},
		{Name: "text_rendering", Status: "not_requested"},
	}

	// When the report renders.
	output := reportTestRender(t, report)

	// Then each stage name appears with its status.
	reportTestRequireLine(t, output, "core_assessment", "complete")
	reportTestRequireLine(t, output, "owner:terraform_binding", "failed")
	reportTestRequireLine(t, output, "composition", "blocked")
	reportTestRequireLine(t, output, "text_rendering", "not_requested")
}

func TestRenderSummaryListsEveryUnresolvedCause(t *testing.T) {
	// Given two unresolved causes.
	report := reportTestBaseReport("source")
	report.Unresolved = []string{
		reportTestBindingCause,
		"The behavior revision is unavailable.",
	}

	// When the report renders.
	output := reportTestRender(t, report)

	// Then each cause appears in the output.
	for _, cause := range report.Unresolved {
		reportTestRequireText(t, output, cause)
	}
}

func TestRenderOwnerShowsStatusAndEachAttachmentOutcomeAndCause(t *testing.T) {
	// Given a complete binding owner with an unknown attachment, a complete view owner with a valid attachment, and a blocked behavior owner.
	report := reportTestBaseReport("source")
	report.Owners = []owners.OwnerResult{
		{
			Extension: model.ID("extension.terraform_binding"),
			Status:    owners.Status("complete"),
			Results: []owners.AttachmentResult{{
				Attachment: "admission_configuration",
				Subject:    model.ID("pv.admission_policy"),
				Outcome:    owners.Outcome("unknown"),
				Cause:      reportTestBindingCause,
			}},
		},
		{
			Extension: model.ID("extension.html_view"),
			Status:    owners.Status("complete"),
			Results: []owners.AttachmentResult{{
				Attachment: "receipt_layout",
				Subject:    model.ID("design.receipt"),
				Outcome:    owners.Outcome("valid"),
			}},
		},
		{
			Extension: model.ID("extension.event_model_link"),
			Status:    owners.Status("blocked"),
		},
	}

	// When the report renders.
	output := reportTestRender(t, report)

	// Then each owner shows its status and each attachment shows its outcome, and the unknown attachment shows its cause.
	reportTestRequireLine(t, output, "terraform_binding", "complete")
	reportTestRequireLine(t, output, "admission_configuration", "unknown")
	reportTestRequireText(t, output, reportTestBindingCause)
	reportTestRequireLine(t, output, "html_view", "complete")
	reportTestRequireLine(t, output, "receipt_layout", "valid")
	reportTestRequireLine(t, output, "event_model_link", "blocked")
}

func TestRenderScopeShowsClassificationAndOrder(t *testing.T) {
	// Given a decoupled sibling scope whose stable order places dp.first_control before dp.second_control.
	report := reportTestBaseReport("source")
	report.Scopes = []assessment.ScopeResult{{
		Matrix:         assessment.Matrix{Scope: model.ID("design.D")},
		Classification: assessment.Classification("decoupled"),
		Order:          []model.ID{"dp.first_control", "dp.second_control"},
	}}

	// When the report renders.
	output := reportTestRender(t, report)

	// Then the scope shows its classification and the controls appear in the given order.
	reportTestRequireLine(t, output, "decoupled")
	first := reportTestPosition(t, output, "dp.first_control")
	second := reportTestPosition(t, output, "dp.second_control")
	if first >= second {
		t.Fatalf("dp.first_control at %d, dp.second_control at %d; want the stated order", first, second)
	}
}

func TestExitCodeIsZeroWithOnlyWarningsAndQualifiedUnknowns(t *testing.T) {
	// Given warning findings, an unresolved external cause, a not requested owner stage, and all required stages complete.
	report := reportTestBaseReport("source")
	report.Stages = append(report.Stages, validate.Stage{Name: "owner:terraform_binding", Status: "not_requested"})
	report.Findings = []result.Finding{
		{Code: "support_gap", Severity: result.Warning, Subject: "dp.alpha", Message: "Alpha claim lacks support."},
		{Code: "unavailable_external_input", Severity: result.Warning, Subject: "pv.admission_policy", Message: "Observation data is unavailable."},
	}
	report.Unresolved = []string{reportTestBindingCause}

	// When the exit code is computed.
	got := ExitCode(report)

	// Then the exit code is 0.
	if got != 0 {
		t.Fatalf("ExitCode() = %d, want 0", got)
	}
}

func TestExitCodeIsOneWithAnErrorFinding(t *testing.T) {
	// Given one error finding and all required stages complete.
	report := reportTestBaseReport("source")
	report.Findings = []result.Finding{
		{Code: "invalid_reference", Severity: result.Error, Subject: "dp.alpha", Message: "Alpha reference is wrong."},
	}

	// When the exit code is computed.
	got := ExitCode(report)

	// Then the exit code is 1.
	if got != 1 {
		t.Fatalf("ExitCode() = %d, want 1", got)
	}
}

func TestExitCodeIsOneWhenRequiredStageFailedOrBlockedWithoutFindings(t *testing.T) {
	cases := []struct {
		name   string
		stage  string
		status string
	}{
		{name: "failed core assessment", stage: "core_assessment", status: "failed"},
		{name: "blocked core assessment", stage: "core_assessment", status: "blocked"},
		{name: "failed required owner", stage: "owner:terraform_binding", status: "failed"},
		{name: "blocked required owner", stage: "owner:terraform_binding", status: "blocked"},
		{name: "failed composition", stage: "composition", status: "failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a required stage with a failed or blocked status and no findings.
			report := reportTestBaseReport("source")
			report.Stages = []validate.Stage{{Name: tc.stage, Status: tc.status}}

			// When the exit code is computed.
			got := ExitCode(report)

			// Then the exit code is 1.
			if got != 1 {
				t.Fatalf("ExitCode() = %d, want 1", got)
			}
		})
	}
}

func TestExitCodeFollowsProfileSeverityForTheSameFinding(t *testing.T) {
	// Given the same shared resource review gap, with severity applied for the source and architecture profiles.
	const code = "shared_resource_review_gap"
	const subject = "pv.admission_policy"
	const message = "The shared resource lacks an influence assessment."
	sourceSeverity, ok := result.SeverityFor(code, result.ProfileSource)
	if !ok {
		t.Fatalf("SeverityFor(%q, source) reports an unknown code", code)
	}
	architectureSeverity, ok := result.SeverityFor(code, result.ProfileArchitecture)
	if !ok {
		t.Fatalf("SeverityFor(%q, architecture) reports an unknown code", code)
	}
	if sourceSeverity != result.Warning || architectureSeverity != result.Error {
		t.Fatalf("severities = %q and %q, want warning and error", sourceSeverity, architectureSeverity)
	}
	sourceReport := reportTestBaseReport("source")
	sourceReport.Findings = []result.Finding{{Code: code, Severity: sourceSeverity, Subject: subject, Message: message}}
	architectureReport := reportTestBaseReport("architecture")
	architectureReport.Findings = []result.Finding{{Code: code, Severity: architectureSeverity, Subject: subject, Message: message}}

	// When each report computes its exit code.
	sourceExit := ExitCode(sourceReport)
	architectureExit := ExitCode(architectureReport)

	// Then the source profile exits 0 and the architecture profile exits 1.
	if sourceExit != 0 {
		t.Fatalf("source ExitCode() = %d, want 0", sourceExit)
	}
	if architectureExit != 1 {
		t.Fatalf("architecture ExitCode() = %d, want 1", architectureExit)
	}
}

func TestRenderReturnsWriterErrorWhenWriterFails(t *testing.T) {
	// Given a complete report and a writer that rejects every write.
	report := reportTestBaseReport("source")
	writerErr := errors.New("output closed")

	// When Render writes the report.
	err := Render(reportTestFailingWriter{err: writerErr}, report)

	// Then Render returns the writer error.
	if !errors.Is(err, writerErr) {
		t.Fatalf("Render() error = %v, want writer error %v", err, writerErr)
	}
}

func TestRenderWritesNothingAfterWriterFails(t *testing.T) {
	// Given a report, its full text, and a writer that accepts only half of that text.
	report := reportTestBaseReport("source")
	report.Findings = []result.Finding{
		{Code: "invalid_reference", Severity: result.Error, Subject: "dp.alpha", Message: "Alpha reference is wrong."},
	}
	full := reportTestRender(t, report)
	writerErr := errors.New("output closed")
	writer := &reportTestLimitedWriter{limit: len(full) / 2, err: writerErr}

	// When Render writes the report.
	err := Render(writer, report)

	// Then Render returns the writer error, makes no write after the failure, and the accepted text is a strict prefix of the full report.
	if !errors.Is(err, writerErr) {
		t.Fatalf("Render() error = %v, want writer error %v", err, writerErr)
	}
	if writer.writesAfterFailure != 0 {
		t.Fatalf("writes after failure = %d, want 0", writer.writesAfterFailure)
	}
	accepted := writer.accepted.String()
	if len(accepted) >= len(full) || !strings.HasPrefix(full, accepted) {
		t.Fatalf("accepted output is not a strict prefix of the full report; accepted %q", accepted)
	}
}
