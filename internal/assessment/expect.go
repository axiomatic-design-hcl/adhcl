package assessment

import (
	"fmt"
	"slices"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
)

// EvaluateExpectations checks every expect record of the graph against the classified
// sibling sets. Each record that does not hold yields one failed_expectation finding on
// its own ID. A record whose under condition does not apply is skipped. Assessment
// leaves the severity empty; composition sets it.
func EvaluateExpectations(snapshot *selection.Snapshot, scopes []ScopeResult) []result.Finding {
	var findings []result.Finding
	for _, expect := range snapshot.Graph.Core.Expects {
		reason := expectFailure(snapshot, scopes, expect)
		if reason == "" {
			continue
		}
		findings = append(findings, expectFinding(expect, reason))
	}
	return findings
}

// expectFailure returns why an expect record does not hold. It returns the empty string
// when the record holds or when its under condition does not apply.
func expectFailure(snapshot *selection.Snapshot, scopes []ScopeResult, expect model.Expect) string {
	switch snapshot.Applies(expect.Under) {
	case selection.Applicability("not_applicable"):
		return ""
	case selection.Applicability("unresolved"):
		return "its under condition is unresolved"
	}
	scope, found := expectScopeOf(scopes, expect.Scope.Target)
	if !found {
		return fmt.Sprintf("scope %s is not a sibling set", expect.Scope.Target)
	}
	if reason := expectClassificationFailure(scope, expect.Classification); reason != "" {
		return reason
	}
	if reason := expectBasisFailure(expect.Basis); reason != "" {
		return reason
	}
	if expect.Order.Present {
		return expectOrderFailure(scope, expect.Order.Items)
	}
	return ""
}

// expectScopeOf finds the classified result of the sibling set that an expect record names.
func expectScopeOf(scopes []ScopeResult, scope model.ID) (ScopeResult, bool) {
	for _, candidate := range scopes {
		if candidate.Matrix.Scope == scope {
			return candidate, true
		}
	}
	return ScopeResult{}, false
}

// expectClassificationFailure explains why the scope does not match the asserted
// classification. A conflict in any cell fails every classification, including unresolved.
func expectClassificationFailure(scope ScopeResult, asserted model.Classification) string {
	if anyCell(scope.Matrix, func(cell Cell) bool { return cell.Conflict }) {
		return fmt.Sprintf("scope %s has conflicting influences, which satisfy no classification", scope.Matrix.Scope)
	}
	if string(scope.Classification) != string(asserted) {
		return fmt.Sprintf("scope %s is %s, not %s", scope.Matrix.Scope, scope.Classification, asserted)
	}
	return ""
}

// expectBasisFailure checks the basis of an assertion. Version 1 has no accepted review
// input, so a supported basis never holds. Source and conditional bases hold when the
// applicable effects establish the structure. Assumed premises stay named conditions and
// do not fail either basis.
func expectBasisFailure(basis *model.ExpectBasis) string {
	if expectBasisOf(basis) == model.ExpectBasisSupported {
		return "its supported basis needs accepted review support, and this version has no review input"
	}
	return ""
}

// expectBasisOf returns the basis of an expect record, where an omitted basis means source.
func expectBasisOf(basis *model.ExpectBasis) model.ExpectBasis {
	if basis == nil {
		return model.ExpectBasisSource
	}
	return *basis
}

// expectOrderFailure checks an authored DP order against the scope. The order must list
// every active DP exactly once, and every known precedence edge must run forward. Unknown
// cells leave precedence unresolved, so they fail the order.
func expectOrderFailure(scope ScopeResult, order []model.Ref) string {
	if anyCell(scope.Matrix, func(cell Cell) bool { return cell.Effect == EffectUnknown }) {
		return fmt.Sprintf("scope %s has unknown cells, so precedence stays unresolved", scope.Matrix.Scope)
	}
	if len(scope.Pairing) != len(scope.Matrix.FRs) {
		return fmt.Sprintf("scope %s has no single complete pairing, so precedence stays unresolved", scope.Matrix.Scope)
	}
	position, reason := expectOrderPositions(scope.Matrix.DPs, order)
	if reason != "" {
		return reason
	}
	return expectPrecedenceFailure(scope, position)
}

// expectOrderPositions maps each listed DP to its position in the order. It fails when the
// order lists a DP that is not active in the scope, repeats one, or omits an active DP.
func expectOrderPositions(activeDPs []model.ID, order []model.Ref) (map[model.ID]int, string) {
	position := make(map[model.ID]int, len(order))
	for index, ref := range order {
		if !slices.Contains(activeDPs, ref.Target) {
			return nil, fmt.Sprintf("its order lists %s, which is not an active DP of the scope", ref.Target)
		}
		if _, repeated := position[ref.Target]; repeated {
			return nil, fmt.Sprintf("its order repeats %s", ref.Target)
		}
		position[ref.Target] = index
	}
	for _, dp := range activeDPs {
		if _, listed := position[dp]; !listed {
			return nil, fmt.Sprintf("its order omits the active DP %s", dp)
		}
	}
	return position, ""
}

// expectPrecedenceFailure reports the first known material effect that runs against the
// order. A DP must come before the DP paired with an FR that it materially affects.
func expectPrecedenceFailure(scope ScopeResult, position map[model.ID]int) string {
	for _, fr := range scope.Matrix.FRs {
		paired := scope.Pairing[fr]
		for _, source := range scope.Matrix.DPs {
			if source == paired || !isKnownMaterial(scope.Matrix.Cell(fr, source)) {
				continue
			}
			if position[paired] < position[source] {
				return fmt.Sprintf("its order places %s before %s, but %s must come first because it materially affects %s", paired, source, source, fr)
			}
		}
	}
	return ""
}

// expectFinding reports an expect record that does not hold.
func expectFinding(expect model.Expect, reason string) result.Finding {
	span := expect.Range
	return result.Finding{
		Code:    result.CodeFailedExpectation,
		Subject: string(expect.ID),
		Message: fmt.Sprintf("Expectation %s does not hold: %s.", expect.ID, reason),
		Range:   &span,
	}
}
