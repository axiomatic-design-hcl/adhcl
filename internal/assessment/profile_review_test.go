package assessment

import (
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
)

func TestReviewCriteriaRetainsEachApplicableQualitativeCriterion(t *testing.T) {
	source := profileTestReplace(t, profileTestBase, "  statement = \"Return the configured left string unchanged.\"\n", "  statement = \"Return the configured left string unchanged.\"\n  criterion \"readable\" { statement = \"The output is readable.\" }\n  criterion \"clear\" { statement = \"The output is clear.\" }\n")
	snapshot := profileTestSnapshot(t, source)
	findings := newProfileChecker(snapshot).criteria()
	profileTestRequireSubjects(t, "incomplete criteria", profileTestSubjects(findings, result.CodeIncompleteCriterion), "fr.left.criterion.readable", "fr.left.criterion.clear")
	for _, finding := range findings {
		if finding.Range == nil || !strings.Contains(finding.Message, finding.Subject) {
			t.Fatalf("criterion finding loses its identity or range: %+v", finding)
		}
		profileReviewRequireSeverity(t, finding.Code)
	}
}

func TestReviewCriteriaRequiresApplicableOperationalCoverage(t *testing.T) {
	for _, selected := range []model.ID{"", "context.other"} {
		t.Run(string(selected), func(t *testing.T) {
			snapshot := profileTestSnapshot(t, profileTestBase)
			criterion := &snapshot.Graph.Core.FRs[0].Criteria[0]
			criterion.Under = &model.Ref{Target: "context.expected"}
			snapshot.Context = selected
			findings := newProfileChecker(snapshot).criteria()
			profileTestRequireSubjects(t, "uncovered FR", profileTestSubjects(findings, result.CodeIncompleteCriterion), "fr.left")
		})
	}
}

func TestReviewCriteriaReportsQualitativePredicateAndMissingCoverage(t *testing.T) {
	snapshot := profileTestSnapshot(t, profileTestBase)
	snapshot.Graph.Core.FRs[0].Criteria[0].Metric = nil
	findings := newProfileChecker(snapshot).criteria()
	profileTestRequireSubjects(t, "incomplete criteria", profileTestSubjects(findings, result.CodeIncompleteCriterion), "fr.left", "fr.left.criterion.exact")
}

func TestReviewCriteriaIgnoresInactiveFRAndInapplicableQualitativeCriterion(t *testing.T) {
	snapshot := profileTestSnapshot(t, profileTestBase)
	fr := &snapshot.Graph.Core.FRs[0]
	qualitative := fr.Criteria[0]
	qualitative.ID = "fr.left.criterion.qualitative"
	qualitative.Metric = nil
	qualitative.Under = &model.Ref{Target: "context.other"}
	fr.Criteria = append(fr.Criteria, qualitative)
	snapshot.Context = "context.selected"
	profileTestRequireNoCode(t, newProfileChecker(snapshot).criteria(), result.CodeIncompleteCriterion)
	fr.Criteria[0].Metric = nil
	snapshot.ActiveFRs[fr.ID] = false
	profileTestRequireNoCode(t, newProfileChecker(snapshot).criteria(), result.CodeIncompleteCriterion)
}

const profileReviewSharedPV = `pv "shared" {
  realizes = [dp.left_output, dp.right_output]
}
`

func TestReviewSharedPVRetainsResourceAndControlIdentities(t *testing.T) {
	snapshot := profileTestSnapshot(t, profileTestBase, profileReviewSharedPV)
	findings := newProfileChecker(snapshot).sharedResourceGaps()
	profileTestRequireSubjects(t, "shared PV", profileTestSubjects(findings, result.CodeSharedResourceReviewGap), "pv.shared")
	finding := findings[0]
	for _, identity := range []string{"pv.shared", "dp.left_output", "dp.right_output"} {
		if !strings.Contains(finding.Message, identity) {
			t.Fatalf("shared PV message %q omits %s", finding.Message, identity)
		}
	}
	if finding.Range == nil || *finding.Range != snapshot.Graph.Core.PVs[0].Range || len(finding.Related) != 2 {
		t.Fatalf("shared PV finding loses resource or control ranges: %+v", finding)
	}
	profileReviewRequireSeverity(t, finding.Code)
}

func TestReviewSharedPVIgnoresInactiveAndCompositeRealizations(t *testing.T) {
	for _, inactive := range []bool{true, false} {
		t.Run(map[bool]string{true: "inactive", false: "composite"}[inactive], func(t *testing.T) {
			snapshot := profileTestSnapshot(t, profileTestBase, profileReviewSharedPV)
			if inactive {
				snapshot.ActiveDPs["dp.right_output"] = false
			} else {
				snapshot.Graph.Children["dp.right_output"] = []model.ID{"fr.left"}
			}
			profileTestRequireNoCode(t, newProfileChecker(snapshot).sharedResourceGaps(), result.CodeSharedResourceReviewGap)
		})
	}
}

func TestReviewSharedPVRequiresApplicableNoninvalidatedAssessment(t *testing.T) {
	for _, state := range []string{"applicable", "inapplicable", "invalidated"} {
		t.Run(state, func(t *testing.T) {
			snapshot := profileTestSnapshot(t, profileTestBase, profileReviewSharedPV, `influence {
  from = [dp.left_output]
  to = fr.right
  effect = "none"
  because = "Separate immutable inputs prevent cross-effects despite shared realization."
}
`)
			applied := &snapshot.Influences[len(snapshot.Influences)-1]
			if state == "inapplicable" {
				applied.Applicability = selection.Applicability("not_applicable")
			}
			if state == "invalidated" {
				applied.Premises = profileInvalidated
			}
			findings := newProfileChecker(snapshot).sharedResourceGaps()
			if state == "applicable" {
				profileTestRequireNoCode(t, findings, result.CodeSharedResourceReviewGap)
			} else {
				profileTestRequireSubjects(t, "shared PV", profileTestSubjects(findings, result.CodeSharedResourceReviewGap), "pv.shared")
			}
		})
	}
}

func profileReviewRequireSeverity(t *testing.T, code string) {
	t.Helper()
	for profile, want := range map[result.Profile]result.Severity{result.ProfileSource: result.Warning, result.ProfileArchitecture: result.Error} {
		if got, ok := result.SeverityFor(code, profile); !ok || got != want {
			t.Fatalf("%s severity under %s = %s, %t; want %s", code, profile, got, ok, want)
		}
	}
}
