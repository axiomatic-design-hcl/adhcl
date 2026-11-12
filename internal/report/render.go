package report

import (
	"fmt"
	"io"

	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
	"github.com/hashicorp/hcl/v2"
)

// closingSentence states what exit code 0 means under the command contract.
const closingSentence = "Exit code 0 means that validation completed under the profile, not that the design is independent or implemented."

// Render writes the report as plain text. It stops at the first write error,
// returns that error, and writes nothing after it.
func Render(w io.Writer, report validate.Result) error {
	out := &lineWriter{out: w}
	writeHeader(out, report)
	writeFindings(out, report.Findings)
	writeScopes(out, report.Scopes)
	writeCandidates(out, report.Candidates)
	writeOwners(out, report.Owners)
	writeSummary(out, report)
	out.line(closingSentence)
	return out.err
}

// lineWriter writes one line at a time and keeps the first write error.
// After an error, later lines are skipped.
type lineWriter struct {
	out io.Writer
	err error
}

func (lw *lineWriter) line(text string) {
	if lw.err != nil {
		return
	}
	_, lw.err = io.WriteString(lw.out, text+"\n")
}

func writeHeader(out *lineWriter, report validate.Result) {
	out.line("Validation report")
	out.line("Input: " + report.Input)
	out.line("Tool version: " + report.ToolVersion)
	out.line("Contract: " + report.Contract)
	out.line("Profile: " + report.Profile)
	out.line("Candidate: " + valueOrDefault(report.Candidate, "default"))
	out.line("Context: " + valueOrDefault(report.Context, "none"))
	out.line("Workload: " + valueOrDefault(report.Workload, "none"))
	out.line("")
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func writeFindings(out *lineWriter, findings []result.Finding) {
	out.line("Findings")
	if len(findings) == 0 {
		out.line("- none.")
	}
	for _, finding := range findings {
		writeFinding(out, finding)
	}
	out.line("")
}

func writeFinding(out *lineWriter, finding result.Finding) {
	out.line(findingText(finding))
	for _, related := range finding.Related {
		out.line("  Related range: " + locationText(related))
	}
}

// findingText names the severity, code, and subject. It adds the location only
// when the finding has a source range.
func findingText(finding result.Finding) string {
	text := fmt.Sprintf("- %s %s on %s", finding.Severity, finding.Code, finding.Subject)
	if finding.Range != nil {
		text += " at " + locationText(*finding.Range)
	}
	return text + ": " + finding.Message
}

func locationText(location hcl.Range) string {
	return fmt.Sprintf("%s:%d:%d", location.Filename, location.Start.Line, location.Start.Column)
}
