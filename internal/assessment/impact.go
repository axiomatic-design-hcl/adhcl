package assessment

import (
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
)

// Impact separates known effects from effects whose assessment is unknown.
type Impact struct {
	DP                                                 string
	Affected, Possible, Needs, Scenarios, Modules, PVs []string
}

// Impacts returns the known and possible effects of each master DP. Needs follow the
// inheritance rule of EffectiveNeeds, and scenarios include those that verify a
// criterion of an affected FR or of one of its ancestor FRs.
func Impacts(snapshot *selection.Snapshot, scopes []ScopeResult) []Impact {
	masterScope := MasterScope(scopes)
	if masterScope == nil {
		return nil
	}
	master := &masterScope.Matrix
	core := &snapshot.Graph.Core
	effective := EffectiveNeeds(core)
	frs := make(map[model.ID]*model.FR, len(core.FRs))
	for i := range core.FRs {
		frs[core.FRs[i].ID] = &core.FRs[i]
	}
	controls := make(map[model.ID][]model.ID, len(core.DPs))
	for _, dp := range core.DPs {
		for _, ref := range dp.Controls {
			controls[dp.ID] = append(controls[dp.ID], ref.Target)
		}
	}
	var impacts []Impact
	for _, dp := range master.DPs {
		impact := Impact{DP: string(dp)}
		affected := make(map[model.ID]bool)
		for _, fr := range master.FRs {
			cell := master.Cell(fr, dp)
			switch cell.Effect {
			case EffectMaterial:
				impact.Affected = append(impact.Affected, string(fr))
				affected[fr] = true
			case EffectUnknown:
				impact.Possible = append(impact.Possible, string(fr))
			}
		}
		needs, criteria := map[model.ID]bool{}, map[model.ID]bool{}
		for _, fr := range master.FRs {
			if !affected[fr] {
				continue
			}
			for _, need := range effective[fr].Needs {
				needs[need] = true
			}
			collectCriteria(fr, frs, controls, criteria, map[model.ID]bool{})
		}
		for _, need := range core.Needs {
			if needs[need.ID] {
				impact.Needs = append(impact.Needs, string(need.ID))
			}
		}
		scenarios := map[model.ID]bool{}
		for _, scenario := range core.Scenarios {
			if referencesAny(scenario.Verifies.Items, criteria) {
				scenarios[scenario.ID] = true
				impact.Scenarios = append(impact.Scenarios, string(scenario.ID))
			}
		}
		for _, module := range core.Modules {
			realized := false
			for _, id := range snapshot.Modules[module.ID] {
				if id == dp {
					realized = true
					break
				}
			}
			if realized || referencesAny(module.AcceptanceChecks.Items, criteria) || referencesAny(module.AcceptanceChecks.Items, scenarios) {
				impact.Modules = append(impact.Modules, string(module.ID))
			}
		}
		for _, pv := range core.PVs {
			for _, ref := range pv.Realizes {
				if ref.Target == dp {
					impact.PVs = append(impact.PVs, string(pv.ID))
					break
				}
			}
		}
		impacts = append(impacts, impact)
	}
	return impacts
}

// collectCriteria adds the criteria of fr and of its ancestor FRs.
func collectCriteria(fr model.ID, frs map[model.ID]*model.FR, controls map[model.ID][]model.ID, criteria, seen map[model.ID]bool) {
	record := frs[fr]
	if record == nil || seen[fr] {
		return
	}
	seen[fr] = true
	for _, criterion := range record.Criteria {
		criteria[criterion.ID] = true
	}
	if record.Parent != nil {
		for _, ancestor := range controls[record.Parent.Target] {
			collectCriteria(ancestor, frs, controls, criteria, seen)
		}
	}
}

func referencesAny(refs []model.Ref, ids map[model.ID]bool) bool {
	for _, ref := range refs {
		if ids[ref.Target] {
			return true
		}
	}
	return false
}
