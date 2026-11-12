package schema

import (
	"fmt"
	"math/big"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
)

// Admitted enum values for the hierarchy blocks.
var (
	lifecycleNames       = []string{string(model.LifecycleProposed), string(model.LifecycleSelected), string(model.LifecycleRejected), string(model.LifecycleSuperseded)}
	effectNames          = []string{string(model.EffectMaterial), string(model.EffectNone), string(model.EffectUnknown)}
	mechanismOriginNames = []string{string(model.MechanismFromSource), string(model.MechanismInferred)}
	unlistedNames        = []string{string(model.UnlistedNone), string(model.UnlistedUnknown)}
	classificationNames  = []string{string(model.ClassificationUncoupled), string(model.ClassificationDecoupled), string(model.ClassificationCoupled), string(model.ClassificationUnresolved)}
	expectBasisNames     = []string{string(model.ExpectBasisSource), string(model.ExpectBasisConditional), string(model.ExpectBasisSupported)}
	criterionOpNames     = []string{string(model.CriterionOpLessEqual), string(model.CriterionOpLess), string(model.CriterionOpGreaterEqual), string(model.CriterionOpGreater), string(model.CriterionOpEqual), string(model.CriterionOpIn)}
)

// decodeFR reads one functional requirement and its nested criterion blocks.
func decodeFR(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindFR), aliasedMeta)
	statement := r.requiredString("statement")
	need := readFRNeed(r)
	parent := r.optionalRef("parent")
	checkRootFRNeed(r)
	boundedBy := r.optionalRefs("bounded_by")
	fr := model.FR{
		Header:    header,
		Statement: statement,
		Need:      need,
		Parent:    parent,
		BoundedBy: boundedBy,
	}
	for _, block := range r.childBlocks(string(model.KindCriterion)) {
		fr.Criteria = append(fr.Criteria, decodeCriterion(r, block, header.ID))
	}
	core.FRs = append(core.FRs, fr)
}

// checkRootFRNeed reports an FR that has neither a parent nor a need. A root FR must name the need it serves.
// An authored attribute counts even when its value is invalid, so a defect is reported only at its own line.
func checkRootFRNeed(r *blockReader) {
	if attributePresent(r, "parent") || attributePresent(r, "need") {
		return
	}
	r.fail(blockHeaderRange(r.block), "an fr with no parent must name the need it serves")
}

// readFRNeed reads fr.need, which accepts one reference or a nonempty list.
// An omitted need is recorded as absent, not as an empty list.
func readFRNeed(r *blockReader) model.Optional[model.Ref] {
	return optionalRefSet(r, "need", oneOrMoreRefsOf)
}

// optionalRefSet reads an optional reference list and records whether it was authored.
func optionalRefSet(r *blockReader, name string, parse func(hclsyntax.Expression) ([]model.Ref, string)) model.Optional[model.Ref] {
	refs, ok := field(r, name, false, parse)
	return model.Optional[model.Ref]{Present: ok, Items: refs}
}

// attributePresent reports whether the block authored the attribute, even if its value is invalid.
func attributePresent(r *blockReader, name string) bool {
	_, ok := r.block.Body.Attributes[name]
	return ok
}

// attributeRange returns the source range of an authored attribute.
func attributeRange(r *blockReader, name string) hcl.Range {
	return r.block.Body.Attributes[name].SrcRange
}

// decodeCriterion reads one criterion nested in an FR.
func decodeCriterion(parent *blockReader, block *hclsyntax.Block, frID model.ID) model.Criterion {
	r := newBlockReader(parent.file, block, parent.findings)
	header := r.namedHeader(string(frID)+"."+string(model.KindCriterion), criterionMeta)
	statement := r.optionalString("statement")
	metric := r.optionalString("metric")
	op := r.optionalEnum("op", criterionOpNames...)
	checkCriterionTriple(r)
	value := readCriterionValue(r, op)
	unit := r.optionalString("unit")
	under := r.optionalRef("under")
	allocatedFrom := r.optionalRef("allocated_from")
	r.finish()
	return model.Criterion{
		Header:        header,
		Statement:     statement,
		Metric:        metric,
		Op:            criterionOpOf(op),
		Value:         value,
		Unit:          unit,
		Under:         under,
		AllocatedFrom: allocatedFrom,
	}
}

// checkCriterionTriple reports a partial operational triple, and a criterion with neither a statement nor a triple.
// It checks authored attributes, so an invalid value does not hide a missing companion.
func checkCriterionTriple(r *blockReader) {
	present := 0
	for _, name := range []string{"metric", "op", "value"} {
		if attributePresent(r, name) {
			present++
		}
	}
	switch {
	case present > 0 && present < 3:
		r.fail(blockHeaderRange(r.block), "an operational criterion needs metric, op, and value together")
	case present == 0 && !attributePresent(r, "statement"):
		r.fail(blockHeaderRange(r.block), "a criterion needs a statement or the complete metric, op, and value")
	}
}

// criterionOpOf converts an optional authored operator to its model type.
func criterionOpOf(op *string) *model.CriterionOp {
	if op == nil {
		return nil
	}
	chosen := model.CriterionOp(*op)
	return &chosen
}

// readCriterionValue reads the value attribute with the shape that its operator fixes.
// Without a valid operator the value cannot be interpreted, so it is only marked as read.
func readCriterionValue(r *blockReader, op *string) *model.CriterionValue {
	if op == nil {
		r.take("value", false)
		return nil
	}
	return optionalField(r, "value", criterionValueParser(model.CriterionOp(*op)))
}

// criterionValueParser returns the value shape for one operator.
func criterionValueParser(op model.CriterionOp) func(hclsyntax.Expression) (model.CriterionValue, string) {
	switch op {
	case model.CriterionOpIn:
		return membershipValueOf
	case model.CriterionOpEqual:
		return equalityValueOf
	default:
		return orderedValueOf
	}
}

// orderedValueOf accepts one number for the comparison operators.
func orderedValueOf(expr hclsyntax.Expression) (model.CriterionValue, string) {
	number, problem := numberOf(expr)
	if problem != "" {
		return model.CriterionValue{}, "must be a number for this comparison"
	}
	return model.CriterionValue{
		Kind:   model.ValueNumber,
		Scalar: model.Scalar{Kind: model.ValueNumber, Number: number},
	}, ""
}

// equalityValueOf accepts one scalar for equality. The scalar keeps its type.
func equalityValueOf(expr hclsyntax.Expression) (model.CriterionValue, string) {
	scalar, problem := criterionScalarOf(expr)
	if problem != "" {
		return model.CriterionValue{}, problem
	}
	return model.CriterionValue{Kind: scalar.Kind, Scalar: scalar}, ""
}

// membershipValueOf accepts a list of scalars or an inclusive interval for membership.
func membershipValueOf(expr hclsyntax.Expression) (model.CriterionValue, string) {
	if object, ok := expr.(*hclsyntax.ObjectConsExpr); ok {
		interval, problem := intervalOf(object)
		if problem != "" {
			return model.CriterionValue{}, problem
		}
		return model.CriterionValue{Kind: model.ValueInterval, Interval: interval}, ""
	}
	list, problem := criterionScalarListOf(expr)
	if problem != "" {
		return model.CriterionValue{}, problem
	}
	return model.CriterionValue{Kind: model.ValueList, List: list}, ""
}

// criterionScalarOf accepts one string, number, or Boolean literal.
func criterionScalarOf(expr hclsyntax.Expression) (model.Scalar, string) {
	const problem = "must be a string, number, or true or false"
	val, ok := literal(expr)
	if !ok {
		return model.Scalar{}, problem
	}
	switch val.Type() {
	case cty.String:
		text, textProblem := stringOf(expr)
		if textProblem != "" {
			return model.Scalar{}, textProblem
		}
		return model.Scalar{Kind: model.ValueString, Text: text}, ""
	case cty.Number:
		number, numberProblem := numberOf(expr)
		if numberProblem != "" {
			return model.Scalar{}, numberProblem
		}
		return model.Scalar{Kind: model.ValueNumber, Number: number}, ""
	case cty.Bool:
		flag, flagProblem := boolOf(expr)
		if flagProblem != "" {
			return model.Scalar{}, flagProblem
		}
		return model.Scalar{Kind: model.ValueBool, Flag: flag}, ""
	}
	return model.Scalar{}, problem
}

// criterionScalarListOf accepts a nonempty list of scalars of one type, with no duplicates, in authored order.
func criterionScalarListOf(expr hclsyntax.Expression) ([]model.Scalar, string) {
	items, ok := listItems(expr)
	if !ok {
		return nil, "must be a list of values or an interval with min and max"
	}
	if len(items) == 0 {
		return nil, "must not be an empty list"
	}
	list := make([]model.Scalar, 0, len(items))
	for _, item := range items {
		scalar, problem := criterionScalarOf(item)
		if problem != "" {
			return nil, problem
		}
		list = append(list, scalar)
	}
	return list, membershipProblem(list)
}

// membershipProblem reports a mixed-type list or a repeated value. It returns "" for a valid list.
func membershipProblem(list []model.Scalar) string {
	for index, scalar := range list {
		if scalar.Kind != list[0].Kind {
			return "must use one kind of value throughout"
		}
		for _, earlier := range list[:index] {
			if sameScalar(earlier, scalar) {
				return "must not repeat a value"
			}
		}
	}
	return ""
}

// sameScalar compares two scalars of the same kind.
func sameScalar(a, b model.Scalar) bool {
	switch a.Kind {
	case model.ValueNumber:
		return a.Number.Cmp(b.Number) == 0
	case model.ValueBool:
		return a.Flag == b.Flag
	default:
		return a.Text == b.Text
	}
}

// intervalOf reads an inclusive interval object with exactly min and max.
func intervalOf(object *hclsyntax.ObjectConsExpr) (model.Interval, string) {
	fields := make(map[string]hclsyntax.Expression, len(object.Items))
	for _, item := range object.Items {
		key, problem := objectKeyOf(item.KeyExpr)
		if problem != "" {
			return model.Interval{}, "interval keys must be min and max"
		}
		if key != "min" && key != "max" {
			return model.Interval{}, fmt.Sprintf("interval has unknown key %q", key)
		}
		if _, repeated := fields[key]; repeated {
			return model.Interval{}, fmt.Sprintf("interval repeats key %q", key)
		}
		fields[key] = item.ValueExpr
	}
	lowest, problem := intervalEnd(fields, "min")
	if problem != "" {
		return model.Interval{}, problem
	}
	highest, problem := intervalEnd(fields, "max")
	if problem != "" {
		return model.Interval{}, problem
	}
	if lowest.Cmp(highest) > 0 {
		return model.Interval{}, "interval min must not be greater than max"
	}
	return model.Interval{Min: lowest, Max: highest}, ""
}

// intervalEnd reads one required endpoint of an interval.
func intervalEnd(fields map[string]hclsyntax.Expression, key string) (*big.Float, string) {
	expr, ok := fields[key]
	if !ok {
		return nil, fmt.Sprintf("interval needs %s", key)
	}
	number, problem := numberOf(expr)
	if problem != "" {
		return nil, fmt.Sprintf("interval %s %s", key, problem)
	}
	return number, ""
}

// decodeDP reads one design parameter.
func decodeDP(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindDP), aliasedMeta)
	controls := r.requiredRefs("controls")
	responsibility := r.requiredString("responsibility")
	decision := model.LifecycleState(r.requiredEnum("decision", lifecycleNames...))
	why := r.optionalString("why")
	role := r.optionalString("role")
	ownsState := optionalRefSet(r, "owns_state", refsOf)
	readsState := r.optionalRefs("reads_state")
	boundedBy := r.optionalRefs("bounded_by")
	holdsIf := r.optionalRefs("holds_if")
	core.DPs = append(core.DPs, model.DP{
		Header:         header,
		Controls:       controls,
		Responsibility: responsibility,
		Decision:       decision,
		Why:            why,
		Role:           role,
		OwnsState:      ownsState,
		ReadsState:     readsState,
		BoundedBy:      boundedBy,
		HoldsIf:        holdsIf,
	})
}

// decodeInfluence reads one unlabeled influence. Each block is one independent claim.
func decodeInfluence(r *blockReader, core *model.Core) {
	header := r.unlabeledHeader(standardMeta)
	from := r.requiredRefs("from")
	to := r.requiredRef("to")
	effect := r.requiredEnum("effect", effectNames...)
	mechanism := r.optionalString("mechanism")
	because := r.optionalString("because")
	mechanismSource := r.optionalEnum("mechanism_source", mechanismOriginNames...)
	source := r.optionalString("source")
	under := r.optionalRef("under")
	holdsIf := r.optionalRefs("holds_if")
	checkInfluenceCompanions(r, effect)
	core.Influences = append(core.Influences, model.Influence{
		Unlabeled:       header,
		From:            from,
		To:              to,
		Effect:          model.Effect(effect),
		Mechanism:       mechanism,
		Because:         because,
		MechanismSource: mechanismOriginOf(mechanismSource),
		SourceLocator:   source,
		Under:           under,
		HoldsIf:         holdsIf,
	})
}

// checkInfluenceCompanions reports a missing mechanism or reason for the claimed effect,
// and a mechanism source that has no mechanism. It checks authored attributes only.
func checkInfluenceCompanions(r *blockReader, effect string) {
	if effect == string(model.EffectMaterial) && !attributePresent(r, "mechanism") {
		r.fail(blockHeaderRange(r.block), "a material influence needs a mechanism")
	}
	if effect == string(model.EffectNone) && !attributePresent(r, "because") {
		r.fail(blockHeaderRange(r.block), "a none influence needs a because reason")
	}
	if attributePresent(r, "mechanism_source") && !attributePresent(r, "mechanism") {
		r.fail(attributeRange(r, "mechanism_source"), "mechanism_source needs a mechanism")
	}
}

// mechanismOriginOf converts an optional authored mechanism source to its model type.
func mechanismOriginOf(source *string) *model.MechanismOrigin {
	if source == nil {
		return nil
	}
	chosen := model.MechanismOrigin(*source)
	return &chosen
}

// decodeCells reads the default effect for unlisted cells of one sibling set.
func decodeCells(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindCells), standardMeta)
	scope := r.requiredRef("scope")
	unlisted := r.requiredEnum("unlisted", unlistedNames...)
	because := r.optionalString("because")
	if unlisted == string(model.UnlistedNone) && !attributePresent(r, "because") {
		r.fail(blockHeaderRange(r.block), `cells with unlisted = "none" need a because reason`)
	}
	core.Cells = append(core.Cells, model.Cells{
		Header:   header,
		Scope:    scope,
		Unlisted: model.UnlistedEffect(unlisted),
		Because:  because,
	})
}

// decodeExpect reads one expected classification of a sibling set. Basis defaults to source.
func decodeExpect(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindExpect), standardMeta)
	scope := r.requiredRef("scope")
	classification := model.Classification(r.requiredEnum("classification", classificationNames...))
	order := optionalRefSet(r, "order", nonEmptyRefsOf)
	basis := model.ExpectBasisSource
	if chosen := r.optionalEnum("basis", expectBasisNames...); chosen != nil {
		basis = model.ExpectBasis(*chosen)
	}
	under := r.optionalRef("under")
	core.Expects = append(core.Expects, model.Expect{
		Header:         header,
		Scope:          scope,
		Classification: classification,
		Order:          order,
		Basis:          &basis,
		Under:          under,
	})
}

// decodeCoupling reads one residual coupling record of a sibling set.
func decodeCoupling(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindCoupling), standardMeta)
	scope := r.requiredRef("scope")
	affects := r.requiredRefs("affects")
	reason := r.requiredString("reason")
	condition := r.optionalString("condition")
	decision := r.optionalString("decision")
	normalPractice := r.optionalBool("normal_practice")
	residualRisk := r.optionalString("residual_risk")
	iterationBound := r.optionalInteger("iteration_bound", true)
	convergence := r.optionalString("convergence")
	stopping := r.optionalString("stopping")
	failureBehavior := r.optionalString("failure_behavior")
	core.Couplings = append(core.Couplings, model.Coupling{
		Header:          header,
		Scope:           scope,
		Affects:         affects,
		Reason:          reason,
		Condition:       condition,
		Decision:        decision,
		NormalPractice:  normalPractice,
		ResidualRisk:    residualRisk,
		IterationBound:  iterationBound,
		Convergence:     convergence,
		Stopping:        stopping,
		FailureBehavior: failureBehavior,
	})
}
