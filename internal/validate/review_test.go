package validate_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/report"
	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

func TestArchitectureRetainsUnresolvedSupport(t *testing.T) {
	r := validate.Run(validate.Request{Path: "../../examples/architecture.ad.hcl", Profile: "architecture"})
	if r.Failed() {
		t.Fatal("architecture example unexpectedly fails")
	}
	if len(r.Unresolved) == 0 {
		t.Fatal("missing support must remain unresolved")
	}
	for _, s := range r.Scopes {
		if s.Classification != "decoupled" {
			t.Fatalf("structural classification changed: %s", s.Classification)
		}
	}
}

func TestOwnerPayloadRevisionChangesWithoutChangingCore(t *testing.T) {
	dir := t.TempDir()
	entries, err := os.ReadDir("../../examples/project")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join("../../examples/project", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, e.Name()), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	req := validate.Request{Path: filepath.Join(dir, "model.ad.project.hcl"), Profile: "source", Version: "test"}
	before := validate.Run(req)
	path := filepath.Join(dir, "views.ad.ext.hcl")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte(strings.Replace(string(b), "matrices_first", "requirements_first", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	after := validate.Run(req)
	if !reflect.DeepEqual(before.Scopes, after.Scopes) {
		t.Fatal("payload changed core conclusions")
	}
	var first, second string
	for _, o := range before.Owners {
		for _, a := range o.Results {
			if a.Attachment == "badge_gate_layout" {
				first = a.Revision
				if a.Range.Filename != path || o.CoreRevision == "" || o.Snapshot == "" || o.Invocation == "" {
					t.Fatal("missing owner provenance")
				}
			}
		}
	}
	for _, o := range after.Owners {
		for _, a := range o.Results {
			if a.Attachment == "badge_gate_layout" {
				second = a.Revision
			}
		}
	}
	if first == "" || first == second {
		t.Fatal("changed payload must change input revision")
	}
}

func TestComposedEvidenceKeepsIndependentQualifiedEstimates(t *testing.T) {
	source := behaviorBaselineModel + `
context "lab" { description = "Fixed test context." }
workload "load" { description = "Fixed requests." }
decision "copy" {
 addresses = [fr.left, fr.right]
 context = context.lab
 workload = workload.load
 horizon = "One request."
 candidate "current" {
  dps = [dp.left_output, dp.right_output]
  version = "v1"
  information {
   basis = "measured"
   evidence = [evidence.first, evidence.second]
  }
 }
}
`
	for _, label := range []string{"first", "second"} {
		source += fmt.Sprintf(`
evidence %q {
 basis = "measured"
 event = "Both outputs match their inputs."
 scenario = scenario.copy_strings
 candidate = decision.copy.candidate.current
 version = "v1"
 context = context.lab
 workload = workload.load
 horizon = "One request."
 dependence = "joint"
 sampling = "Independent requests under fixed inputs."
 trials = 20
 failures = 0
 confidence_method = "wilson"
 confidence = 0.95
}
`, label)
	}
	path := filepath.Join(t.TempDir(), "evidence.ad.hcl")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	r := validate.Run(validate.Request{Path: path, Profile: "source", Candidate: "decision.copy.candidate.current"})
	if r.Failed() {
		t.Fatalf("validation failed: %+v", r.Findings)
	}
	if len(r.Candidates) != 1 || len(r.Candidates[0].Evidence) != 2 {
		t.Fatalf("estimates lost in composition: %+v", r.Candidates)
	}
	a, b := r.Candidates[0].Evidence[0], r.Candidates[0].Evidence[1]
	if a.Evidence == b.Evidence || a.Trials != 20 || a.Failures != 0 || a.Confidence != 0.95 || a.Sampling == "" || a.Dependence != "joint" || a.Version != "v1" || a.Horizon != "One request." || a.Context != "context.lab" || a.Workload != "workload.load" || a.Scenario != "scenario.copy_strings" {
		t.Fatalf("lost estimate qualifications: %+v %+v", a, b)
	}
	if !(a.Low > 0 && a.Low < 1 && a.High == 1) || a.Worst.Status != "finite" || a.Worst.Value <= 0 {
		t.Fatalf("finite sample uncertainty lost: %+v", a)
	}
	var text bytes.Buffer
	if err := report.Render(&text, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text.String(), "evidence.first: 20 trials, 0 failures") || !strings.Contains(text.String(), "evidence.second: 20 trials, 0 failures") || !strings.Contains(text.String(), "Joint success probability (Wilson)") {
		t.Fatal("estimates missing from report")
	}
}

func TestDuplicateAttachmentLabelsSurviveInvalidEnvelopeAndCore(t *testing.T) {
	for _, brokenCore := range []bool{false, true} {
		t.Run(fmt.Sprint(brokenCore), func(t *testing.T) {
			dir := t.TempDir()
			entries, err := os.ReadDir("../../examples/project")
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				b, err := os.ReadFile(filepath.Join("../../examples/project", e.Name()))
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(dir, e.Name()), b, 0600); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(dir, "views.ad.ext.hcl")
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			b = append(b, []byte(`
attachment "badge_gate_layout" {
 owner = extension.html_view
 subject = design.badge_gate
}
`)...)
			if err = os.WriteFile(path, b, 0600); err != nil {
				t.Fatal(err)
			}
			if brokenCore {
				path := filepath.Join(dir, "design.ad.hcl")
				b, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				b = append(b, []byte(`
dp "missing_reference" {
 controls = [fr.missing]
 responsibility = "Invalid reference independent of attachments."
 decision = "selected"
}
`)...)
				if err = os.WriteFile(path, b, 0600); err != nil {
					t.Fatal(err)
				}
			}
			r := validate.Run(validate.Request{Path: filepath.Join(dir, "model.ad.project.hcl"), Profile: "source"})
			var duplicate, invalid bool
			for _, f := range r.Findings {
				if f.Code == "duplicate_identity" && f.Subject == "badge_gate_layout" {
					duplicate = f.Range != nil && len(f.Related) == 1 && f.Range.Start.Line != f.Related[0].Start.Line
				}
				if f.Code == "invalid_attachment" {
					invalid = true
				}
			}
			if !duplicate || !invalid || !r.Failed() {
				t.Fatalf("independent attachment diagnostics lost: %+v", r.Findings)
			}
		})
	}
}
