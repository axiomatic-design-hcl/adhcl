package assessment

import (
	"fmt"
	"strings"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
	"github.com/hashicorp/hcl/v2"
)

const (
	profileApplies     = selection.Applicability("applies")
	profileInvalidated = selection.PremiseState("invalidated")
)

// profileChecker holds the lookups that the profile checks share over one snapshot.
type profileChecker struct {
	snapshot  *selection.Snapshot
	core      *model.Core
	frs       map[model.ID]*model.FR
	dps       map[model.ID]*model.DP
	scenarios map[model.ID]*model.Scenario
}

func newProfileChecker(snapshot *selection.Snapshot) *profileChecker {
	core := &snapshot.Graph.Core
	c := &profileChecker{
		snapshot:  snapshot,
		core:      core,
		frs:       make(map[model.ID]*model.FR, len(core.FRs)),
		dps:       make(map[model.ID]*model.DP, len(core.DPs)),
		scenarios: make(map[model.ID]*model.Scenario, len(core.Scenarios)),
	}
	for i := range core.FRs {
		c.frs[core.FRs[i].ID] = &core.FRs[i]
	}
	for i := range core.DPs {
		c.dps[core.DPs[i].ID] = &core.DPs[i]
	}
	for i := range core.Scenarios {
		c.scenarios[core.Scenarios[i].ID] = &core.Scenarios[i]
	}
	return c
}

// Completeness reports the profile obligations that the selected snapshot leaves unmet:
// missing operational criteria, incomplete module contracts, leaf requirements without a
// verifying scenario, feedback cycles without a coupling record, and shared resources
// without an influence assessment. Findings leave their severity empty.
func Completeness(snapshot *selection.Snapshot, scopes []ScopeResult) []result.Finding {
	c := newProfileChecker(snapshot)
	var findings []result.Finding
	findings = append(findings, c.criteria()...)
	findings = append(findings, c.moduleContracts()...)
	findings = append(findings, c.missingScenarios()...)
	findings = append(findings, c.feedbackContracts(scopes)...)
	findings = append(findings, c.sharedResourceGaps()...)
	return findings
}

// SupportGaps reports applicable influence claims and the premises they rely on that have no
// reviewed support. A claim with its own premises or source premises reports those premises
// instead of itself. Each premise is reported once.
func SupportGaps(snapshot *selection.Snapshot) []result.Finding {
	c := newProfileChecker(snapshot)
	var findings []result.Finding
	reported := make(map[model.ID]bool)
	for _, applied := range snapshot.Influences {
		if applied.Applicability != profileApplies || applied.Premises == profileInvalidated {
			continue
		}
		premises := c.premisesOf(applied)
		if len(premises) == 0 {
			findings = append(findings, profileInfluenceGap(applied))
			continue
		}
		for _, ref := range premises {
			if reported[ref.Target] {
				continue
			}
			reported[ref.Target] = true
			findings = append(findings, profilePremiseGap(ref, applied))
		}
	}
	return findings
}

// applies reports whether a condition holds in the selected snapshot.
func (c *profileChecker) applies(under *model.Ref) bool {
	return c.snapshot.Applies(under) == profileApplies
}

// activeLeafFR reports whether an FR is active and no controlling DP has an active child FR.
func (c *profileChecker) activeLeafFR(fr *model.FR) bool {
	if !c.snapshot.ActiveFRs[fr.ID] {
		return false
	}
	for _, dp := range c.snapshot.Graph.Controllers[fr.ID] {
		if c.hasActiveChild(dp) {
			return false
		}
	}
	return true
}

// activeLeafDP reports whether a DP is active and has no active child FR.
func (c *profileChecker) activeLeafDP(dp *model.DP) bool {
	return c.snapshot.ActiveDPs[dp.ID] && !c.hasActiveChild(dp.ID)
}

// hasActiveChild reports whether a DP has at least one active child FR.
func (c *profileChecker) hasActiveChild(dp model.ID) bool {
	for _, child := range c.snapshot.Graph.Children[dp] {
		if c.snapshot.ActiveFRs[child] {
			return true
		}
	}
	return false
}

// criteria reports incomplete applicable criteria and active FRs without an applicable
// operational predicate. One complete predicate does not complete another criterion.
func (c *profileChecker) criteria() []result.Finding {
	var findings []result.Finding
	for i := range c.core.FRs {
		fr := &c.core.FRs[i]
		if !c.snapshot.ActiveFRs[fr.ID] {
			continue
		}
		hasOperational := false
		for j := range fr.Criteria {
			criterion := &fr.Criteria[j]
			if !c.applies(criterion.Under) {
				continue
			}
			if profileOperational(criterion) {
				hasOperational = true
				continue
			}
			message := fmt.Sprintf("Criterion %s has no complete metric, operator, and value, so its acceptance condition remains qualitative.", criterion.ID)
			findings = append(findings, profileFinding(result.CodeIncompleteCriterion, string(criterion.ID), message, criterion.Range))
		}
		if !hasOperational {
			message := fmt.Sprintf("Requirement %s has no applicable criterion with a metric, operator, and value, so no operational predicate checks it.", fr.ID)
			findings = append(findings, profileFinding(result.CodeIncompleteCriterion, string(fr.ID), message, fr.Range))
		}
	}
	return findings
}

// profileOperational reports whether a criterion carries a metric, an operator, and a value.
func profileOperational(criterion *model.Criterion) bool {
	return criterion.Metric != nil && strings.TrimSpace(*criterion.Metric) != "" &&
		criterion.Op != nil && criterion.Value != nil
}

// moduleContracts reports each active leaf DP whose realization module contract is incomplete.
func (c *profileChecker) moduleContracts() []result.Finding {
	var findings []result.Finding
	for i := range c.core.DPs {
		dp := &c.core.DPs[i]
		if !c.activeLeafDP(dp) {
			continue
		}
		problem := c.moduleProblem(dp)
		if problem == "" {
			continue
		}
		findings = append(findings, profileFinding(result.CodeIncompleteModuleContract, string(dp.ID), problem, dp.Range))
	}
	return findings
}

// moduleProblem names the first missing obligation of a leaf DP's realization module, or
// returns the empty string when the module contract is complete.
func (c *profileChecker) moduleProblem(dp *model.DP) string {
	modules := c.realizationModules(dp.ID)
	if len(modules) == 0 {
		return fmt.Sprintf("Design parameter %s has no realization module, so nothing realizes it.", dp.ID)
	}
	if len(modules) > 1 {
		return fmt.Sprintf("Design parameter %s has %d realization modules, but it needs exactly one.", dp.ID, len(modules))
	}
	module := modules[0]
	if field := profileMissingModuleField(module); field != "" {
		return fmt.Sprintf("Realization module %s of design parameter %s is missing %s.", module.ID, dp.ID, field)
	}
	if criterion, ok := c.uncoveredCriterion(dp, module); ok {
		return fmt.Sprintf("Realization module %s does not check criterion %s, directly or through a scenario that verifies it.", module.ID, criterion)
	}
	return ""
}

// realizationModules lists the modules that realize a DP, in authored order.
func (c *profileChecker) realizationModules(dp model.ID) []*model.Module {
	var modules []*model.Module
	for i := range c.core.Modules {
		module := &c.core.Modules[i]
		if profileRefsContain(module.Realizes.Items, dp) {
			modules = append(modules, module)
		}
	}
	return modules
}

// profileMissingModuleField names the first required module field that is absent, or returns
// the empty string when all required fields are present.
func profileMissingModuleField(module *model.Module) string {
	switch {
	case !module.Inputs.Present:
		return "inputs"
	case !module.Outputs.Present:
		return "outputs"
	case !module.OwnsState.Present:
		return "owns_state"
	case !module.Dependencies.Present:
		return "dependencies"
	case len(module.Invariants.Items) == 0:
		return "invariants"
	case !profileFilled(module.FailureBehavior):
		return "failure_behavior"
	case len(module.AcceptanceChecks.Items) == 0:
		return "acceptance_checks"
	}
	return ""
}

// uncoveredCriterion returns the first applicable criterion of a leaf FR that the module's
// acceptance checks do not cover.
func (c *profileChecker) uncoveredCriterion(dp *model.DP, module *model.Module) (model.ID, bool) {
	for _, ref := range dp.Controls {
		fr := c.frs[ref.Target]
		if fr == nil || !c.activeLeafFR(fr) {
			continue
		}
		for i := range fr.Criteria {
			criterion := &fr.Criteria[i]
			if c.applies(criterion.Under) && !c.checksCover(module.AcceptanceChecks.Items, criterion.ID) {
				return criterion.ID, true
			}
		}
	}
	return "", false
}

// checksCover reports whether an acceptance check names a criterion directly, or names an
// applicable scenario that verifies it.
func (c *profileChecker) checksCover(checks []model.Ref, criterion model.ID) bool {
	for _, check := range checks {
		if check.Target == criterion {
			return true
		}
		scenario := c.scenarios[check.Target]
		if scenario != nil && c.applies(scenario.Under) && profileRefsContain(scenario.Verifies.Items, criterion) {
			return true
		}
	}
	return false
}

// missingScenarios reports each active leaf FR without an applicable scenario that has steps
// and verifies all of its applicable criteria.
func (c *profileChecker) missingScenarios() []result.Finding {
	var findings []result.Finding
	for i := range c.core.FRs {
		fr := &c.core.FRs[i]
		if !c.activeLeafFR(fr) || c.hasCoveringScenario(fr) {
			continue
		}
		message := fmt.Sprintf("Leaf requirement %s has no applicable scenario with steps that verifies all of its applicable criteria.", fr.ID)
		findings = append(findings, profileFinding(result.CodeMissingLeafScenario, string(fr.ID), message, fr.Range))
	}
	return findings
}

// hasCoveringScenario reports whether an applicable scenario with steps verifies every
// applicable criterion of an FR.
func (c *profileChecker) hasCoveringScenario(fr *model.FR) bool {
	criteria := c.applicableCriteria(fr)
	for i := range c.core.Scenarios {
		scenario := &c.core.Scenarios[i]
		if len(scenario.Steps.Items) == 0 || !c.applies(scenario.Under) {
			continue
		}
		if profileVerifiesAll(criteria, scenario.Verifies.Items) {
			return true
		}
	}
	return false
}

// applicableCriteria lists the criteria of an FR that hold in the selected snapshot.
func (c *profileChecker) applicableCriteria(fr *model.FR) []model.ID {
	var criteria []model.ID
	for i := range fr.Criteria {
		criterion := &fr.Criteria[i]
		if c.applies(criterion.Under) {
			criteria = append(criteria, criterion.ID)
		}
	}
	return criteria
}

// feedbackContracts reports each scope with a feedback cycle that no complete coupling record
// covers. Each scope is reported once.
func (c *profileChecker) feedbackContracts(scopes []ScopeResult) []result.Finding {
	var findings []result.Finding
	for _, scope := range scopes {
		cycle, ok := c.uncoveredCycle(scope)
		if !ok {
			continue
		}
		message := fmt.Sprintf("Scope %s has a feedback cycle without a complete coupling record, so its iteration bound, convergence, stopping rule, or failure behavior is missing.", scope.Matrix.Scope)
		findings = append(findings, profileFinding(result.CodeIncompleteFeedbackContract, string(scope.Matrix.Scope), message, c.dpRanges(cycle)...))
	}
	return findings
}

// uncoveredCycle returns the first cycle of a scope that no complete coupling record covers.
func (c *profileChecker) uncoveredCycle(scope ScopeResult) ([]model.ID, bool) {
	for _, cycle := range scope.Cycles {
		if !c.cycleCovered(scope.Matrix.Scope, cycle, scope.Pairing) {
			return cycle, true
		}
	}
	return nil, false
}

// cycleCovered reports whether a complete coupling record of the scope affects every DP of the
// cycle and every FR paired with those DPs.
func (c *profileChecker) cycleCovered(scope model.ID, cycle []model.ID, pairing map[model.ID]model.ID) bool {
	required := profileCycleMembers(cycle, pairing)
	for i := range c.core.Couplings {
		coupling := &c.core.Couplings[i]
		if coupling.Scope.Target != scope || !profileCouplingComplete(coupling) {
			continue
		}
		if profileAffectsAll(coupling.Affects, required) {
			return true
		}
	}
	return false
}

// dpRanges returns the declaration ranges of the DPs in a cycle that are declared.
func (c *profileChecker) dpRanges(cycle []model.ID) []hcl.Range {
	var ranges []hcl.Range
	for _, id := range cycle {
		if dp := c.dps[id]; dp != nil {
			ranges = append(ranges, dp.Range)
		}
	}
	return ranges
}

// sharedResourceGaps reports states and PVs shared by active leaf DPs without an
// applicable influence between them. Each resource is reported once.
func (c *profileChecker) sharedResourceGaps() []result.Finding {
	var findings []result.Finding
	for i := range c.core.States {
		state := &c.core.States[i]
		users := c.leafUsers(state.ID)
		message, ok := c.unreviewedPair("State", state.ID, users)
		if !ok {
			continue
		}
		findings = append(findings, profileFinding(result.CodeSharedResourceReviewGap, string(state.ID), message, state.Range))
	}
	for i := range c.core.PVs {
		pv := &c.core.PVs[i]
		var users []*model.DP
		for j := range c.core.DPs {
			dp := &c.core.DPs[j]
			if c.activeLeafDP(dp) && profileRefsContain(pv.Realizes, dp.ID) {
				users = append(users, dp)
			}
		}
		message, ok := c.unreviewedPair("Process variable", pv.ID, users)
		if !ok {
			continue
		}
		ranges := make([]hcl.Range, 1, len(users)+1)
		ranges[0] = pv.Range
		for _, dp := range users {
			ranges = append(ranges, dp.Range)
		}
		findings = append(findings, profileFinding(result.CodeSharedResourceReviewGap, string(pv.ID), message, ranges...))
	}
	return findings
}

// leafUsers lists the active leaf DPs that own or read a state, in authored order.
func (c *profileChecker) leafUsers(state model.ID) []*model.DP {
	var users []*model.DP
	for i := range c.core.DPs {
		dp := &c.core.DPs[i]
		if c.activeLeafDP(dp) && profileUsesState(dp, state) {
			users = append(users, dp)
		}
	}
	return users
}

// unreviewedPair returns a message for the first pair of users with no applicable influence
// between them, or false when every pair has one.
func (c *profileChecker) unreviewedPair(kind string, resource model.ID, users []*model.DP) (string, bool) {
	for i := range users {
		for j := i + 1; j < len(users); j++ {
			if c.influenceBetween(users[i], users[j]) {
				continue
			}
			message := fmt.Sprintf("%s %s is shared by design parameters %s and %s, but no applicable influence between them is assessed.", kind, resource, users[i].ID, users[j].ID)
			return message, true
		}
	}
	return "", false
}

// influenceBetween reports whether an applicable, not invalidated influence runs from one DP to
// an FR that the other DP controls, in either direction.
func (c *profileChecker) influenceBetween(a, b *model.DP) bool {
	for _, applied := range c.snapshot.Influences {
		if applied.Applicability != profileApplies || applied.Premises == profileInvalidated {
			continue
		}
		if applied.From == a.ID && profileControls(b, applied.To) {
			return true
		}
		if applied.From == b.ID && profileControls(a, applied.To) {
			return true
		}
	}
	return false
}

// premisesOf lists the premises that an applied influence relies on: its own holds_if and the
// holds_if of its source DP.
func (c *profileChecker) premisesOf(applied selection.AppliedInfluence) []model.Ref {
	premises := append([]model.Ref{}, applied.Record.HoldsIf...)
	if dp := c.dps[applied.From]; dp != nil {
		premises = append(premises, dp.HoldsIf...)
	}
	return premises
}

// profileInfluenceGap reports an applied influence claim that has no premises and no reviewed
// support. The subject is the source file and line span of the authored block.
func profileInfluenceGap(applied selection.AppliedInfluence) result.Finding {
	record := applied.Record
	subject := fmt.Sprintf("%s:%d-%d", record.File, record.Range.Start.Line, record.Range.End.Line)
	message := fmt.Sprintf("The claimed influence from %s to %s has no reviewed support.", applied.From, applied.To)
	return profileFinding(result.CodeSupportGap, subject, message, record.Range)
}

// profilePremiseGap reports an assumption that an applied influence relies on without reviewed
// support. The influence range is related when it differs from the assumption range.
func profilePremiseGap(ref model.Ref, applied selection.AppliedInfluence) result.Finding {
	message := fmt.Sprintf("Assumption %s is relied on by a claimed influence but has no reviewed support.", ref.Target)
	return profileFinding(result.CodeSupportGap, string(ref.Target), message, ref.Range, applied.Record.Range)
}

// profileFinding builds a finding with its subject and related ranges. The first range is the
// primary range. Later ranges are related only when they differ from the primary range.
// Severity is left empty for composition to set.
func profileFinding(code, subject, message string, ranges ...hcl.Range) result.Finding {
	finding := result.Finding{Code: code, Subject: subject, Message: message}
	if len(ranges) == 0 {
		return finding
	}
	primary := ranges[0]
	finding.Range = &primary
	for _, rng := range ranges[1:] {
		if rng != primary {
			finding.Related = append(finding.Related, rng)
		}
	}
	return finding
}

// profileFilled reports whether an optional authored string is present and not blank.
func profileFilled(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}

// profileCouplingComplete reports whether a coupling record states its iteration bound,
// convergence, stopping rule, and failure behavior.
func profileCouplingComplete(coupling *model.Coupling) bool {
	return coupling.IterationBound != nil &&
		profileFilled(coupling.Convergence) &&
		profileFilled(coupling.Stopping) &&
		profileFilled(coupling.FailureBehavior)
}

// profileCycleMembers lists the DPs of a cycle and the FRs paired with them.
func profileCycleMembers(cycle []model.ID, pairing map[model.ID]model.ID) []model.ID {
	members := append([]model.ID{}, cycle...)
	for fr, dp := range pairing {
		if profileContains(cycle, dp) {
			members = append(members, fr)
		}
	}
	return members
}

// profileAffectsAll reports whether the affected references include every required ID.
func profileAffectsAll(affects []model.Ref, required []model.ID) bool {
	for _, id := range required {
		if !profileRefsContain(affects, id) {
			return false
		}
	}
	return true
}

// profileVerifiesAll reports whether the verified references include every criterion.
func profileVerifiesAll(criteria []model.ID, verifies []model.Ref) bool {
	for _, criterion := range criteria {
		if !profileRefsContain(verifies, criterion) {
			return false
		}
	}
	return true
}

// profileUsesState reports whether a DP owns or reads a state.
func profileUsesState(dp *model.DP, state model.ID) bool {
	return profileRefsContain(dp.OwnsState.Items, state) || profileRefsContain(dp.ReadsState, state)
}

// profileControls reports whether a DP controls an FR.
func profileControls(dp *model.DP, fr model.ID) bool {
	return profileRefsContain(dp.Controls, fr)
}

// profileRefsContain reports whether any reference targets the ID.
func profileRefsContain(refs []model.Ref, id model.ID) bool {
	for _, ref := range refs {
		if ref.Target == id {
			return true
		}
	}
	return false
}

// profileContains reports whether the list holds the ID.
func profileContains(ids []model.ID, id model.ID) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}
