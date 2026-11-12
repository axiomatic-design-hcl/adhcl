package result

import "testing"

// severityTestRow is one diagnostic code with its catalog severity in each profile.
type severityTestRow struct {
	code         string
	source       Severity
	architecture Severity
}

// severityTestCatalog mirrors the severity columns of testdata/spec/docs/diagnostics.md.
var severityTestCatalog = []severityTestRow{
	{"input_failure", Error, Error},
	{"invalid_syntax", Error, Error},
	{"invalid_structure", Error, Error},
	{"duplicate_identity", Error, Error},
	{"invalid_reference", Error, Error},
	{"invalid_hierarchy", Error, Error},
	{"invalid_selection", Error, Error},
	{"conflicting_influence", Error, Error},
	{"ambiguous_pairing", Error, Error},
	{"uncontrolled_requirement", Error, Error},
	{"redundant_control", Error, Error},
	{"ineffective_control", Error, Error},
	{"conflicting_ownership", Error, Error},
	{"infeasible_candidate", Error, Error},
	{"invalid_evidence", Error, Error},
	{"failed_expectation", Error, Error},
	{"invalid_attachment", Error, Error},
	{"unavailable_owner", Error, Error},
	{"owner_failure", Error, Error},
	{"duplicate_binding", Error, Error},
	{"binding_mismatch", Error, Error},
	{"behavior_target_missing", Error, Error},
	{"render_failure", Error, Error},
	{"incomplete_criterion", Warning, Error},
	{"incomplete_module_contract", Warning, Error},
	{"missing_leaf_scenario", Warning, Error},
	{"incomplete_feedback_contract", Warning, Error},
	{"unresolved_refinement", Warning, Error},
	{"unresolved_applicability", Warning, Error},
	{"unresolved_influence", Warning, Error},
	{"shared_resource_review_gap", Warning, Error},
	{"support_gap", Warning, Warning},
	{"unresolved_feasibility", Warning, Warning},
	{"incomparable_evidence", Warning, Warning},
	{"unavailable_external_input", Warning, Warning},
}

func TestSeverityForReturnsCatalogSeverityForEveryCode(t *testing.T) {
	for _, row := range severityTestCatalog {
		t.Run(row.code, func(t *testing.T) {
			// Given a diagnostic code from the catalog
			code := row.code

			// When its severity is requested for the source and architecture profiles
			sourceSeverity, sourceFound := SeverityFor(code, ProfileSource)
			architectureSeverity, architectureFound := SeverityFor(code, ProfileArchitecture)

			// Then both lookups are found and return the catalog severities
			if !sourceFound {
				t.Fatalf("SeverityFor(%q, source) found = false, want true", code)
			}
			if sourceSeverity != row.source {
				t.Errorf("SeverityFor(%q, source) = %q, want %q", code, sourceSeverity, row.source)
			}
			if !architectureFound {
				t.Fatalf("SeverityFor(%q, architecture) found = false, want true", code)
			}
			if architectureSeverity != row.architecture {
				t.Errorf("SeverityFor(%q, architecture) = %q, want %q", code, architectureSeverity, row.architecture)
			}
		})
	}
}

func TestSeverityForReportsUnknownCodeAsNotFound(t *testing.T) {
	// Given a code that is not in the diagnostics catalog
	const code = "not_a_catalog_code"

	// When its severity is requested for each profile
	_, sourceFound := SeverityFor(code, ProfileSource)
	_, architectureFound := SeverityFor(code, ProfileArchitecture)

	// Then neither profile reports the code as found
	if sourceFound {
		t.Errorf("SeverityFor(%q, source) found = true, want false", code)
	}
	if architectureFound {
		t.Errorf("SeverityFor(%q, architecture) found = true, want false", code)
	}
}
