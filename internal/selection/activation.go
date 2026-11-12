package selection

import (
	"fmt"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

// activeDPs returns the DPs that the choice activates. A nil choice gives the default snapshot.
func (ix *index) activeDPs(choice *candidateChoice) (map[model.ID]bool, []result.Finding) {
	if choice == nil {
		return ix.selectedDPs()
	}
	return ix.candidateDPs(choice)
}

// selectedDPs activates every DP whose lifecycle is selected. It also checks that each
// decision's selected label agrees with the lifecycle of its candidates' DPs.
func (ix *index) selectedDPs() (map[model.ID]bool, []result.Finding) {
	active := make(map[model.ID]bool)
	for i := range ix.core.DPs {
		dp := &ix.core.DPs[i]
		if dp.Decision == model.LifecycleSelected {
			active[dp.ID] = true
		}
	}
	var findings []result.Finding
	for i := range ix.core.Decisions {
		findings = append(findings, ix.checkSelectedLabel(&ix.core.Decisions[i])...)
	}
	return active, findings
}

// checkSelectedLabel requires exactly the selected candidate's DPs to be selected.
// Alternatives within the same decision may share those DPs.
func (ix *index) checkSelectedLabel(decision *model.Decision) []result.Finding {
	var findings []result.Finding
	selected := selectedCandidateID(decision)
	if decision.Selected != nil {
		if _, ok := ix.candidates[selected]; !ok {
			findings = append(findings, invalidFinding(
				string(decision.ID),
				&decision.Range,
				fmt.Sprintf("Decision %s selects %s, which is not one of its candidates.", decision.ID, *decision.Selected),
			))
		}
	}
	selectedDPs := make(map[model.ID]bool)
	if choice := ix.candidates[selected]; choice != nil {
		for _, ref := range choice.candidate.DPs {
			selectedDPs[ref.Target] = true
		}
	}
	for i := range decision.Candidates {
		candidate := &decision.Candidates[i]
		findings = append(findings, ix.checkCandidateLifecycle(candidate, selectedDPs)...)
	}
	return findings
}

// selectedCandidateID returns the ID of the candidate that the decision's selected label
// names, or the empty ID when the decision selects none.
func selectedCandidateID(decision *model.Decision) model.ID {
	if decision.Selected == nil {
		return ""
	}
	return model.ID(string(decision.ID) + ".candidate." + *decision.Selected)
}

// checkCandidateLifecycle checks each DP against the decision's selected candidate membership.
func (ix *index) checkCandidateLifecycle(candidate *model.Candidate, selectedDPs map[model.ID]bool) []result.Finding {
	var findings []result.Finding
	for i := range candidate.DPs {
		ref := &candidate.DPs[i]
		selected := selectedDPs[ref.Target]
		state := ix.dps[ref.Target].Decision
		if selected && state != model.LifecycleSelected {
			findings = append(findings, invalidFinding(
				string(ref.Target),
				&ref.Range,
				fmt.Sprintf("Candidate %s lists DP %s, which belongs to the selected candidate but is %s and not selected.", candidate.ID, ref.Target, state),
			))
		}
		if !selected && state == model.LifecycleSelected {
			findings = append(findings, invalidFinding(
				string(ref.Target),
				&ref.Range,
				fmt.Sprintf("Candidate %s lists DP %s, which is selected, but this candidate is not the selected one.", candidate.ID, ref.Target),
			))
		}
	}
	return findings
}

// candidateDPs activates the chosen candidate's DPs and every common DP. A common DP is
// selected and listed by no candidate of any decision.
func (ix *index) candidateDPs(choice *candidateChoice) (map[model.ID]bool, []result.Finding) {
	active := make(map[model.ID]bool)
	for i := range ix.core.DPs {
		dp := &ix.core.DPs[i]
		if dp.Decision == model.LifecycleSelected && ix.dpDecisions[dp.ID] == "" {
			active[dp.ID] = true
		}
	}
	var findings []result.Finding
	for i := range choice.candidate.DPs {
		ref := &choice.candidate.DPs[i]
		state := ix.dps[ref.Target].Decision
		if state == model.LifecycleRejected || state == model.LifecycleSuperseded {
			findings = append(findings, invalidFinding(
				string(ref.Target),
				&ref.Range,
				fmt.Sprintf("Candidate %s lists DP %s, which is %s and cannot be selected.", choice.candidate.ID, ref.Target, state),
			))
			continue
		}
		active[ref.Target] = true
	}
	return active, findings
}

// activeFRs activates each root FR, and each child FR whose parent DP is active.
func (ix *index) activeFRs(activeDPs map[model.ID]bool) map[model.ID]bool {
	active := make(map[model.ID]bool)
	for i := range ix.core.FRs {
		fr := &ix.core.FRs[i]
		if fr.Parent == nil || activeDPs[fr.Parent.Target] {
			active[fr.ID] = true
		}
	}
	return active
}

// checkClosure reports each active DP that controls an FR left inactive by a missing ancestor.
// It never inserts the missing ancestor.
func (ix *index) checkClosure(activeDPs, activeFRs map[model.ID]bool) []result.Finding {
	var findings []result.Finding
	for i := range ix.core.DPs {
		dp := &ix.core.DPs[i]
		if !activeDPs[dp.ID] {
			continue
		}
		for j := range dp.Controls {
			ref := &dp.Controls[j]
			if !activeFRs[ref.Target] {
				findings = append(findings, invalidFinding(
					string(dp.ID),
					&ref.Range,
					fmt.Sprintf("DP %s is active, but the FR %s it controls is not active because its parent DP is not active.", dp.ID, ref.Target),
				))
			}
		}
	}
	return findings
}

// activeModules maps each module that realizes at least one active DP to those DPs, in
// authored order. Modules with no active realized DP are omitted.
func (ix *index) activeModules(activeDPs map[model.ID]bool) map[model.ID][]model.ID {
	modules := make(map[model.ID][]model.ID)
	for i := range ix.core.Modules {
		module := &ix.core.Modules[i]
		var realized []model.ID
		for _, ref := range module.Realizes.Items {
			if activeDPs[ref.Target] {
				realized = append(realized, ref.Target)
			}
		}
		if len(realized) > 0 {
			modules[module.ID] = realized
		}
	}
	return modules
}

// appliedInfluences expands each influence into one record per source DP whose DP and target
// FR are both active. Each record keeps the authored influence unchanged.
func (ix *index) appliedInfluences(snapshot *Snapshot) []AppliedInfluence {
	var applied []AppliedInfluence
	for i := range ix.core.Influences {
		influence := &ix.core.Influences[i]
		if !snapshot.ActiveFRs[influence.To.Target] {
			continue
		}
		for _, from := range influence.From {
			if !snapshot.ActiveDPs[from.Target] {
				continue
			}
			applied = append(applied, AppliedInfluence{
				Record:        influence,
				From:          from.Target,
				To:            influence.To.Target,
				Applicability: snapshot.Applies(influence.Under),
				Premises:      ix.premiseState(influence, from.Target),
			})
		}
	}
	return applied
}

// premiseState summarizes the premises of one expanded influence: its own holds_if and the
// holds_if of its source DP.
func (ix *index) premiseState(influence *model.Influence, source model.ID) PremiseState {
	sourcePremises := ix.dps[source].HoldsIf
	if len(influence.HoldsIf) == 0 && len(sourcePremises) == 0 {
		return premisesNone
	}
	if ix.anyInvalidated(influence.HoldsIf) || ix.anyInvalidated(sourcePremises) {
		return premisesInvalidated
	}
	return premisesConditional
}

// anyInvalidated reports whether any referenced assumption has the invalidated status.
// An authored validated status stays conditional because v1 has no trusted review input.
func (ix *index) anyInvalidated(refs []model.Ref) bool {
	for _, ref := range refs {
		assumption := ix.assumptions[ref.Target]
		if assumption != nil && assumption.Status != nil && *assumption.Status == model.AssumptionInvalidated {
			return true
		}
	}
	return false
}
