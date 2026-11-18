package diagram

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

const (
	columnCount      = 5
	columnStep       = 230
	nodeWidth        = 190
	nodeHeight       = 96
	rowStart         = 48
	rowStep          = 136
	ruleOffset       = 20
	mapBottomPadding = 80
	emptyMapHeight   = 120
	maxIDRunes       = 23
	maxLineRunes     = 26
)

// TraceMap uses fixed integer coordinates so the page works without script.
type TraceMap struct {
	Width, Height         int
	NodeWidth, NodeHeight int
	Columns               [columnCount]int
	RuleStart, RuleEnd    int
	Nodes                 []Node
	Links                 []Link
	Rules                 []int
}
type Node struct {
	ID, DisplayID, Kind, Text, Line1, Line2 string
	X, Y, Row                               int
	Selectable                              bool
}
type Link struct {
	From, To, Path, Kind, Label string
	LabelX, LabelY              int
}

// buildTraceMap places the leaf FRs of the master matrix with their needs, DPs,
// modules, PVs, and scenarios. A need links to each leaf FR that has it as an
// effective need, so inherited needs show on the map.
func buildTraceMap(res validate.Result, impacts []assessment.Impact, needs map[model.ID]assessment.NeedTrace) TraceMap {
	trace := TraceMap{Width: columnCount * columnStep, NodeWidth: nodeWidth, NodeHeight: nodeHeight, RuleStart: columnStep, RuleEnd: 2*columnStep + nodeWidth}
	for column := range trace.Columns {
		trace.Columns[column] = column * columnStep
	}
	master := assessment.MasterScope(res.Scopes)
	if master == nil {
		trace.Height = emptyMapHeight
		return trace
	}
	positions, next := placeNodes(&trace, res, master, impacts, needs)
	routeLinks(&trace, res, master, positions, needs)
	trace.Height = traceMapHeight(next)
	return trace
}

func traceMapHeight(columnRows [columnCount]int) int {
	tallestColumn := slices.Max(columnRows[:])
	return mapBottomPadding + tallestColumn*rowStep
}

func placeNodes(trace *TraceMap, res validate.Result, master *assessment.ScopeResult, impacts []assessment.Impact, needs map[model.ID]assessment.NeedTrace) (map[string]Node, [columnCount]int) {
	core := res.Snapshot.Graph.Core
	rows := map[string]int{}
	next := [columnCount]int{}
	occupied := [columnCount]map[int]bool{}
	for column := range occupied {
		occupied[column] = map[int]bool{}
	}
	positions := map[string]Node{}
	add := func(column int, id, kind, text string, row int, selectable bool) {
		if _, ok := positions[id]; ok {
			return
		}
		for occupied[column][row] {
			row++
		}
		occupied[column][row] = true
		if row >= next[column] {
			next[column] = row + 1
		}
		line1, line2 := wrap(text)
		display := id
		if runes := []rune(id); len(runes) > maxIDRunes {
			display = string(runes[:maxIDRunes-1]) + "…"
		}
		node := Node{ID: id, DisplayID: display, Kind: kind, Text: text, Line1: line1, Line2: line2, X: column * columnStep, Y: rowStart + row*rowStep, Row: row, Selectable: selectable}
		trace.Nodes = append(trace.Nodes, node)
		positions[id] = node
	}
	grouped, boundaries := groupedLeaves(master.Matrix.FRs, res.Snapshot.Graph)
	for _, row := range boundaries {
		trace.Rules = append(trace.Rules, rowStart+row*rowStep-ruleOffset)
	}
	for i, id := range grouped {
		rows[string(id)] = i
		for _, f := range core.FRs {
			if f.ID == id {
				add(1, string(id), "FR", f.Statement, i, false)
				break
			}
		}
	}
	for _, id := range master.Matrix.DPs {
		row := next[2]
		if len(master.Pairing) == 0 && row < len(master.Matrix.FRs) {
			row = len(master.Matrix.FRs)
		}
		for fr, dp := range master.Pairing {
			if dp == id {
				row = rows[string(fr)]
				break
			}
		}
		for _, d := range core.DPs {
			if d.ID == id {
				add(2, string(id), "DP", d.Responsibility, row, true)
				break
			}
		}
	}
	for _, need := range core.Needs {
		row := next[0]
		for _, leaf := range grouped {
			if slices.Contains(needs[leaf].Needs, need.ID) {
				row = rows[string(leaf)]
				break
			}
		}
		add(0, string(need.ID), "Need", need.Statement, row, false)
	}
	for _, module := range core.Modules {
		if _, active := res.Snapshot.Modules[module.ID]; !active {
			continue
		}
		row := next[3]
		for _, dp := range res.Snapshot.Modules[module.ID] {
			if position, ok := positions[string(dp)]; ok {
				row = position.Row
				break
			}
		}
		add(3, string(module.ID), "Module", "", row, false)
	}
	for _, variable := range core.PVs {
		row := next[3]
		linked := false
		for _, ref := range variable.Realizes {
			if position, ok := positions[string(ref.Target)]; ok {
				row = position.Row
				linked = true
				break
			}
		}
		if linked {
			add(3, string(variable.ID), "PV", text(variable.Note), row, false)
		}
	}
	for _, scenario := range core.Scenarios {
		row := next[4]
		for _, impact := range impacts {
			if slices.Contains(impact.Scenarios, string(scenario.ID)) {
				if position, ok := positions[impact.DP]; ok {
					row = position.Row
					break
				}
			}
		}
		add(4, string(scenario.ID), "Scenario", text(scenario.Description), row, false)
	}
	return positions, next
}

func routeLinks(trace *TraceMap, res validate.Result, master *assessment.ScopeResult, positions map[string]Node, needs map[model.ID]assessment.NeedTrace) {
	core := res.Snapshot.Graph.Core
	link := func(from, to, kind, label string) {
		source, sourceExists := positions[from]
		target, targetExists := positions[to]
		if !sourceExists || !targetExists {
			return
		}
		x1, y1 := source.X+nodeWidth, source.Y+nodeHeight/2
		x2, y2 := target.X, target.Y+nodeHeight/2
		if kind == "influence" {
			x1 = source.X
			x2 = target.X + nodeWidth
		} else if kind == "dependency" {
			x1, x2 = source.X+nodeWidth/2, target.X+nodeWidth/2
			if source.Y < target.Y {
				y1, y2 = source.Y+nodeHeight, target.Y
			} else {
				y1, y2 = source.Y, target.Y+nodeHeight
			}
		}
		var path string
		if y1 == y2 {
			path = fmt.Sprintf("M %d %d L %d %d", x1, y1, x2, y2)
		} else {
			mid := (x1 + x2) / 2
			path = fmt.Sprintf("M %d %d C %d %d %d %d %d %d", x1, y1, mid, y1, mid, y2, x2, y2)
		}
		trace.Links = append(trace.Links, Link{From: from, To: to, Path: path, Kind: kind, Label: label, LabelX: (x1 + x2) / 2, LabelY: (y1 + y2) / 2})
	}
	for _, fr := range master.Matrix.FRs {
		for _, need := range needs[fr].Needs {
			link(string(need), string(fr), "trace", "")
		}
	}
	for _, fr := range master.Matrix.FRs {
		for _, dp := range master.Matrix.DPs {
			cell := master.Matrix.Cell(fr, dp)
			if master.Pairing[fr] == dp {
				link(string(fr), string(dp), "trace", "")
			} else if cell.Effect == assessment.EffectMaterial {
				link(string(dp), string(fr), "influence", "")
			} else if cell.Effect == assessment.EffectUnknown {
				link(string(dp), string(fr), "influence", "?")
			}
		}
	}
	for _, module := range core.Modules {
		for _, dp := range res.Snapshot.Modules[module.ID] {
			link(string(dp), string(module.ID), "trace", "")
		}
	}
	for _, variable := range core.PVs {
		for _, ref := range variable.Realizes {
			link(string(ref.Target), string(variable.ID), "trace", "")
		}
	}
	for _, module := range core.Modules {
		for _, dependency := range module.Dependencies.Items {
			link(string(dependency.Target), string(module.ID), "dependency", "")
		}
		for _, scenario := range core.Scenarios {
			if containsRef(module.AcceptanceChecks.Items, scenario.ID) || sharedRef(module.AcceptanceChecks.Items, scenario.Verifies.Items) {
				link(string(module.ID), string(scenario.ID), "trace", "")
			}
		}
	}
}

func groupedLeaves(leaves []model.ID, graph *model.Graph) ([]model.ID, []int) {
	groupOf := make(map[model.ID]model.ID, len(leaves))
	for _, set := range graph.SiblingSets {
		for _, fr := range set.FRs {
			groupOf[fr] = set.Scope
		}
	}
	var groups []model.ID
	for _, fr := range leaves {
		scope := groupOf[fr]
		if !slices.Contains(groups, scope) {
			groups = append(groups, scope)
		}
	}
	ordered := make([]model.ID, 0, len(leaves))
	var boundaries []int
	for _, group := range groups {
		if len(ordered) > 0 {
			boundaries = append(boundaries, len(ordered))
		}
		for _, fr := range leaves {
			if groupOf[fr] == group {
				ordered = append(ordered, fr)
			}
		}
	}
	return ordered, boundaries
}
func containsRef(refs []model.Ref, id model.ID) bool {
	for _, r := range refs {
		if r.Target == id {
			return true
		}
	}
	return false
}
func sharedRef(a, b []model.Ref) bool {
	for _, r := range a {
		if containsRef(b, r.Target) {
			return true
		}
	}
	return false
}
func wrap(s string) (string, string) {
	words := strings.Fields(s)
	lines := []string{"", ""}
	row := 0
	for _, w := range words {
		candidate := w
		if lines[row] != "" {
			candidate = lines[row] + " " + w
		}
		if utf8.RuneCountInString(candidate) > maxLineRunes && lines[row] != "" {
			row++
			if row == 2 {
				lines[1] += "…"
				break
			}
			candidate = w
		}
		if utf8.RuneCountInString(candidate) > maxLineRunes {
			r := []rune(candidate)
			candidate = string(r[:maxLineRunes-1]) + "…"
		}
		lines[row] = candidate
	}
	return lines[0], lines[1]
}
