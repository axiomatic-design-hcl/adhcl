package assessment

import (
	"slices"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
)

// NeedTrace is the effective needs of one FR. From is empty when the FR states its
// own need. Otherwise it names the nearest ancestor FRs that state the inherited needs.
type NeedTrace struct {
	Needs []model.ID
	From  []model.ID
}

// Inherited reports whether the needs come from ancestor FRs.
func (t NeedTrace) Inherited() bool { return len(t.From) > 0 }

// EffectiveNeeds applies the schema rule for every FR: a child without need inherits
// the union of needs from the FRs that its parent DP controls. Needs keep the authored
// order of the need blocks, and From keeps the authored order of the FR blocks.
func EffectiveNeeds(core *model.Core) map[model.ID]NeedTrace {
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
	needOrder := make(map[model.ID]int, len(core.Needs))
	for i, need := range core.Needs {
		needOrder[need.ID] = i
	}
	frOrder := make(map[model.ID]int, len(core.FRs))
	for i, fr := range core.FRs {
		frOrder[fr.ID] = i
	}
	traces := make(map[model.ID]NeedTrace, len(core.FRs))
	visiting := make(map[model.ID]bool)
	var resolve func(model.ID) NeedTrace
	resolve = func(id model.ID) NeedTrace {
		if trace, done := traces[id]; done {
			return trace
		}
		fr := frs[id]
		if fr == nil || visiting[id] {
			return NeedTrace{}
		}
		var trace NeedTrace
		if len(fr.Need.Items) > 0 {
			for _, ref := range fr.Need.Items {
				trace.Needs = appendUnique(trace.Needs, ref.Target)
			}
		} else if fr.Parent != nil {
			visiting[id] = true
			for _, sibling := range controls[fr.Parent.Target] {
				parent := resolve(sibling)
				for _, need := range parent.Needs {
					trace.Needs = appendUnique(trace.Needs, need)
				}
				if parent.Inherited() {
					for _, from := range parent.From {
						trace.From = appendUnique(trace.From, from)
					}
				} else if len(parent.Needs) > 0 {
					trace.From = appendUnique(trace.From, sibling)
				}
			}
			delete(visiting, id)
		}
		slices.SortStableFunc(trace.Needs, func(a, b model.ID) int { return needOrder[a] - needOrder[b] })
		slices.SortStableFunc(trace.From, func(a, b model.ID) int { return frOrder[a] - frOrder[b] })
		traces[id] = trace
		return trace
	}
	for _, fr := range core.FRs {
		resolve(fr.ID)
	}
	return traces
}

func appendUnique(ids []model.ID, id model.ID) []model.ID {
	if slices.Contains(ids, id) {
		return ids
	}
	return append(ids, id)
}

// Junction is the composition rule that a classification gives to the modules of
// one sibling set: S (summation) for uncoupled, C (control) for decoupled, and
// F (feedback) for coupled. An unresolved scope has no junction.
func Junction(classification Classification) string {
	switch classification {
	case Uncoupled:
		return "S"
	case Decoupled:
		return "C"
	case Coupled:
		return "F"
	}
	return ""
}
