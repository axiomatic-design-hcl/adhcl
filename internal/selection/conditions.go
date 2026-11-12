package selection

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

// conditions are the effective values that one selection resolves.
type conditions struct {
	context  model.ID
	workload model.ID
	horizon  string
}

// Applies reports whether a condition reference holds for this snapshot. A nil reference
// always applies. A reference to a kind that has no selected value is unresolved.
func (s *Snapshot) Applies(under *model.Ref) Applicability {
	if under == nil {
		return applicabilityApplies
	}
	switch under.Target.Kind() {
	case model.KindContext:
		return compareCondition(under.Target, s.Context)
	case model.KindWorkload:
		return compareCondition(under.Target, s.Workload)
	default:
		return applicabilityUnresolved
	}
}

// compareCondition compares a referenced condition with the selected value of its kind.
func compareCondition(target, selected model.ID) Applicability {
	switch {
	case selected == "":
		return applicabilityUnresolved
	case target == selected:
		return applicabilityApplies
	default:
		return applicabilityNotApplicable
	}
}

// resolveConditions resolves the effective context, workload, and horizon for a selection.
func (ix *index) resolveConditions(choice *candidateChoice, requested model.ID) (conditions, []result.Finding) {
	declaredContext, declaredWorkload, horizon := declaredConditions(choice)
	context, findings := ix.effectiveContext(choice, declaredContext, requested)
	if len(findings) > 0 {
		return conditions{}, findings
	}
	workload, findings := ix.effectiveWorkload(choice, declaredWorkload, context)
	return conditions{context: context, workload: workload, horizon: horizon}, findings
}

// declaredConditions returns the context, workload, and horizon that the candidate declares,
// falling back to the values its decision declares when the candidate is silent.
func declaredConditions(choice *candidateChoice) (model.ID, model.ID, string) {
	if choice == nil {
		return "", "", ""
	}
	candidate, decision := choice.candidate, choice.decision
	context := firstID(refTarget(candidate.Context), refTarget(decision.Context))
	workload := firstID(refTarget(candidate.Workload), refTarget(decision.Workload))
	horizon := firstString(candidate.Horizon, decision.Horizon)
	return context, workload, horizon
}

// effectiveContext combines the declared context with the request. An empty declared context
// takes the request, and a request that differs from the declared context is a conflict.
func (ix *index) effectiveContext(choice *candidateChoice, declared, requested model.ID) (model.ID, []result.Finding) {
	if requested == "" {
		return declared, nil
	}
	if ix.contexts[requested] == nil {
		return "", []result.Finding{
			invalidFinding(string(requested), nil, fmt.Sprintf("The requested context %s is not declared.", requested)),
		}
	}
	if declared == "" || declared == requested {
		return requested, nil
	}
	return "", []result.Finding{ix.contextConflict(choice, declared, requested)}
}

// contextConflict reports a requested context that cannot be selected with the declared one.
func (ix *index) contextConflict(choice *candidateChoice, declared, requested model.ID) result.Finding {
	message := fmt.Sprintf("The requested context %s differs from the declared context %s.", requested, declared)
	if ix.excludesEither(declared, requested) {
		message = fmt.Sprintf("Contexts %s and %s cannot both be selected because one excludes the other.", declared, requested)
	}
	subject, span := conflictSite(choice, requested)
	return invalidFinding(subject, span, message)
}

// excludesEither reports whether either context excludes the other.
func (ix *index) excludesEither(a, b model.ID) bool {
	return excludes(ix.contexts[a], b) || excludes(ix.contexts[b], a)
}

// excludes reports whether a context lists the given ID in its excludes.
func excludes(context *model.Context, id model.ID) bool {
	if context == nil {
		return false
	}
	for _, ref := range context.Excludes {
		if ref.Target == id {
			return true
		}
	}
	return false
}

// effectiveWorkload applies the workload fallbacks. With a candidate, the candidate's declared
// workload must agree with its context's workload, and the design root is the last fallback.
// Without a candidate, the context's workload must agree with the design root.
func (ix *index) effectiveWorkload(choice *candidateChoice, declared, context model.ID) (model.ID, []result.Finding) {
	fromContext := ix.contextWorkload(context)
	design := ix.designWorkload()
	if choice == nil {
		if disagree(fromContext, design) {
			return "", []result.Finding{workloadConflict(choice, context, fromContext, design)}
		}
		return firstID(fromContext, design), nil
	}
	if disagree(declared, fromContext) {
		return "", []result.Finding{workloadConflict(choice, context, declared, fromContext)}
	}
	return firstID(declared, fromContext, design), nil
}

// disagree reports whether two workloads are both known and different.
func disagree(a, b model.ID) bool {
	return a != "" && b != "" && a != b
}

// workloadConflict reports two workloads that cannot both hold for the selection.
func workloadConflict(choice *candidateChoice, context, a, b model.ID) result.Finding {
	subject, span := conflictSite(choice, context)
	return invalidFinding(subject, span, fmt.Sprintf("Workload %s disagrees with workload %s.", a, b))
}

// conflictSite names the selected candidate when there is one, because its authored range is
// the place where the conflict starts. Otherwise it names the fallback subject.
func conflictSite(choice *candidateChoice, fallback model.ID) (string, *hcl.Range) {
	if choice == nil {
		return string(fallback), nil
	}
	return string(choice.candidate.ID), &choice.candidate.Range
}

// contextWorkload returns the workload that a context declares, or the empty ID.
func (ix *index) contextWorkload(context model.ID) model.ID {
	found := ix.contexts[context]
	if found == nil {
		return ""
	}
	return refTarget(found.Workload)
}

// designWorkload returns the workload that the design root declares, or the empty ID.
func (ix *index) designWorkload() model.ID {
	if len(ix.core.Designs) == 0 {
		return ""
	}
	return refTarget(ix.core.Designs[0].Workload)
}
