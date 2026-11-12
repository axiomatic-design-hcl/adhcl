package result

// Profile selects which severity column of the diagnostic catalog applies.
type Profile string

const (
	ProfileSource       Profile = "source"
	ProfileArchitecture Profile = "architecture"
)

// Diagnostic rule codes shared by every package that reports findings.
const (
	CodeInputFailure               = "input_failure"
	CodeInvalidSyntax              = "invalid_syntax"
	CodeInvalidStructure           = "invalid_structure"
	CodeDuplicateIdentity          = "duplicate_identity"
	CodeInvalidReference           = "invalid_reference"
	CodeInvalidHierarchy           = "invalid_hierarchy"
	CodeInvalidSelection           = "invalid_selection"
	CodeConflictingInfluence       = "conflicting_influence"
	CodeAmbiguousPairing           = "ambiguous_pairing"
	CodeUncontrolledRequirement    = "uncontrolled_requirement"
	CodeRedundantControl           = "redundant_control"
	CodeIneffectiveControl         = "ineffective_control"
	CodeConflictingOwnership       = "conflicting_ownership"
	CodeIncompleteCriterion        = "incomplete_criterion"
	CodeIncompleteModuleContract   = "incomplete_module_contract"
	CodeMissingLeafScenario        = "missing_leaf_scenario"
	CodeIncompleteFeedbackContract = "incomplete_feedback_contract"
	CodeUnresolvedRefinement       = "unresolved_refinement"
	CodeUnresolvedApplicability    = "unresolved_applicability"
	CodeUnresolvedInfluence        = "unresolved_influence"
	CodeSharedResourceReviewGap    = "shared_resource_review_gap"
	CodeSupportGap                 = "support_gap"
	CodeUnresolvedFeasibility      = "unresolved_feasibility"
	CodeInfeasibleCandidate        = "infeasible_candidate"
	CodeInvalidEvidence            = "invalid_evidence"
	CodeIncomparableEvidence       = "incomparable_evidence"
	CodeFailedExpectation          = "failed_expectation"
	CodeInvalidAttachment          = "invalid_attachment"
	CodeUnavailableOwner           = "unavailable_owner"
	CodeOwnerFailure               = "owner_failure"
	CodeDuplicateBinding           = "duplicate_binding"
	CodeBindingMismatch            = "binding_mismatch"
	CodeBehaviorTargetMissing      = "behavior_target_missing"
	CodeUnavailableExternalInput   = "unavailable_external_input"
	CodeRenderFailure              = "render_failure"
)

// severityPair holds the catalog severity of one code in each profile.
type severityPair struct {
	source       Severity
	architecture Severity
}

// severityByCode mirrors the severity columns of testdata/spec/docs/diagnostics.md.
var severityByCode = map[string]severityPair{
	CodeInputFailure:               {source: Error, architecture: Error},
	CodeInvalidSyntax:              {source: Error, architecture: Error},
	CodeInvalidStructure:           {source: Error, architecture: Error},
	CodeDuplicateIdentity:          {source: Error, architecture: Error},
	CodeInvalidReference:           {source: Error, architecture: Error},
	CodeInvalidHierarchy:           {source: Error, architecture: Error},
	CodeInvalidSelection:           {source: Error, architecture: Error},
	CodeConflictingInfluence:       {source: Error, architecture: Error},
	CodeAmbiguousPairing:           {source: Error, architecture: Error},
	CodeUncontrolledRequirement:    {source: Error, architecture: Error},
	CodeRedundantControl:           {source: Error, architecture: Error},
	CodeIneffectiveControl:         {source: Error, architecture: Error},
	CodeConflictingOwnership:       {source: Error, architecture: Error},
	CodeIncompleteCriterion:        {source: Warning, architecture: Error},
	CodeIncompleteModuleContract:   {source: Warning, architecture: Error},
	CodeMissingLeafScenario:        {source: Warning, architecture: Error},
	CodeIncompleteFeedbackContract: {source: Warning, architecture: Error},
	CodeUnresolvedRefinement:       {source: Warning, architecture: Error},
	CodeUnresolvedApplicability:    {source: Warning, architecture: Error},
	CodeUnresolvedInfluence:        {source: Warning, architecture: Error},
	CodeSharedResourceReviewGap:    {source: Warning, architecture: Error},
	CodeSupportGap:                 {source: Warning, architecture: Warning},
	CodeUnresolvedFeasibility:      {source: Warning, architecture: Warning},
	CodeInfeasibleCandidate:        {source: Error, architecture: Error},
	CodeInvalidEvidence:            {source: Error, architecture: Error},
	CodeIncomparableEvidence:       {source: Warning, architecture: Warning},
	CodeFailedExpectation:          {source: Error, architecture: Error},
	CodeInvalidAttachment:          {source: Error, architecture: Error},
	CodeUnavailableOwner:           {source: Error, architecture: Error},
	CodeOwnerFailure:               {source: Error, architecture: Error},
	CodeDuplicateBinding:           {source: Error, architecture: Error},
	CodeBindingMismatch:            {source: Error, architecture: Error},
	CodeBehaviorTargetMissing:      {source: Error, architecture: Error},
	CodeUnavailableExternalInput:   {source: Warning, architecture: Warning},
	CodeRenderFailure:              {source: Error, architecture: Error},
}

// SeverityFor returns the catalog severity of code under profile.
// It reports false when the code or the profile is unknown.
func SeverityFor(code string, profile Profile) (Severity, bool) {
	pair, found := severityByCode[code]
	if !found {
		return "", false
	}
	switch profile {
	case ProfileSource:
		return pair.source, true
	case ProfileArchitecture:
		return pair.architecture, true
	default:
		return "", false
	}
}
