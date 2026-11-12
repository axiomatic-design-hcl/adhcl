package schema

import (
	"fmt"
	"math/big"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

// Diagnostic codes used by the realization and manifest decoders.
const (
	codeInvalidEvidence   = "invalid_evidence"
	codeInvalidAttachment = "invalid_attachment"
	codeDuplicateIdentity = "duplicate_identity"
)

// addCodedFinding appends one error finding with an explicit diagnostic code.
func addCodedFinding(findings *[]result.Finding, code, subject string, rng hcl.Range, message string) {
	*findings = append(*findings, result.Finding{
		Code:     code,
		Severity: result.Error,
		Subject:  subject,
		Message:  message,
		Range:    &rng,
	})
}

// failCoded reports one finding with an explicit diagnostic code for this block.
func (r *blockReader) failCoded(code string, rng hcl.Range, message string) {
	addCodedFinding(r.findings, code, r.subject, rng, message)
}

// optionalList reads an optional list field. Present reflects whether the
// attribute was written, even when its value is rejected.
func optionalList[T any](r *blockReader, name string, parse func(hclsyntax.Expression) ([]T, string)) model.Optional[T] {
	items, _ := field(r, name, false, parse)
	return model.Optional[T]{Present: attributePresent(r, name), Items: items}
}

// selectedLabelOf parses the label of a candidate selected by a decision.
func selectedLabelOf(expr hclsyntax.Expression) (string, string) {
	label, problem := stringOf(expr)
	if problem != "" {
		return "", problem
	}
	if !labelPattern.MatchString(label) {
		return "", "must match the label grammar: lower case letters, digits, and single underscores"
	}
	return label, ""
}

// decodeDecision decodes one decision block and its candidate blocks.
func decodeDecision(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindDecision), standardMeta)
	addresses := r.requiredRefs("addresses")
	context := r.optionalRef("context")
	workload := r.optionalRef("workload")
	horizon := r.optionalString("horizon")
	selected := optionalField(r, "selected", selectedLabelOf)
	provisional := true
	if declared := r.optionalBool("provisional"); declared != nil {
		provisional = *declared
	}
	why := r.optionalString("why")

	blocks := r.childBlocks(string(model.KindCandidate))
	if len(blocks) == 0 {
		r.fail(blockHeaderRange(r.block), "a decision needs at least one candidate block")
	}
	candidates := make([]model.Candidate, 0, len(blocks))
	for _, block := range blocks {
		candidates = append(candidates, decodeCandidate(r, block, header.ID))
	}

	core.Decisions = append(core.Decisions, model.Decision{
		Header:      header,
		Addresses:   addresses,
		Context:     context,
		Workload:    workload,
		Horizon:     horizon,
		Selected:    selected,
		Provisional: &provisional,
		Why:         why,
		Candidates:  candidates,
	})
}

// decodeCandidate decodes one candidate block nested in a decision.
func decodeCandidate(parent *blockReader, block *hclsyntax.Block, decisionID model.ID) model.Candidate {
	r := newBlockReader(parent.file, block, parent.findings)
	header := r.namedHeader(string(decisionID)+"."+string(model.KindCandidate), standardMeta)
	dps := r.requiredRefs("dps")
	version := r.optionalString("version")
	context := r.optionalRef("context")
	workload := r.optionalRef("workload")
	horizon := r.optionalString("horizon")

	var information *model.Information
	for i, child := range r.childBlocks("information") {
		if i > 0 {
			r.fail(blockHeaderRange(child), "a candidate may contain at most one information block")
			continue
		}
		information = decodeInformation(r, child)
	}
	r.finish()

	return model.Candidate{
		Header:      header,
		DPs:         dps,
		Version:     version,
		Context:     context,
		Workload:    workload,
		Horizon:     horizon,
		Information: information,
	}
}

// decodeInformation decodes the unlabeled information block of a candidate.
func decodeInformation(parent *blockReader, block *hclsyntax.Block) *model.Information {
	r := newBlockReader(parent.file, block, parent.findings)
	header := r.unlabeledHeader(standardMeta)
	basis := model.InformationBasis(r.requiredEnum("basis", string(model.InformationUnknown), string(model.InformationMeasured)))
	evidence, parsed := field(r, "evidence", false, refsOf)
	if basis == model.InformationMeasured {
		switch {
		case !attributePresent(r, "evidence"):
			r.fail(blockHeaderRange(block), "a measured information block needs at least one evidence reference")
		case parsed && len(evidence) == 0:
			r.fail(attributeRange(r, "evidence"), "evidence must list at least one reference when basis is measured")
		}
	}
	r.finish()

	return &model.Information{Unlabeled: header, Basis: basis, Evidence: evidence}
}

// decodeEvidence decodes one evidence block. Every field is required.
func decodeEvidence(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindEvidence), standardMeta)
	basis := model.EvidenceBasis(r.requiredEnum("basis", string(model.EvidenceMeasured)))
	event := r.requiredString("event")
	scenario := r.requiredRef("scenario")
	candidate := r.requiredRef("candidate")
	version := r.requiredString("version")
	context := r.requiredRef("context")
	workload := r.requiredRef("workload")
	horizon := r.requiredString("horizon")
	dependence := model.Dependence(r.requiredEnum("dependence", string(model.DependenceJoint)))
	sampling := r.requiredString("sampling")
	trials, failures := readTrialCounts(r)
	confidenceMethod := model.ConfidenceMethod(r.requiredEnum("confidence_method", string(model.ConfidenceWilson)))
	confidence := r.requiredNumber("confidence")

	core.Evidence = append(core.Evidence, model.Evidence{
		Header:           header,
		Basis:            basis,
		Event:            event,
		Scenario:         scenario,
		Candidate:        candidate,
		Version:          version,
		Context:          context,
		Workload:         workload,
		Horizon:          horizon,
		Dependence:       dependence,
		Sampling:         sampling,
		Trials:           trials,
		Failures:         failures,
		ConfidenceMethod: confidenceMethod,
		Confidence:       confidence,
	})
}

// readTrialCounts reads the trial and failure counts of an evidence block.
// Type errors are structural; count violations are invalid evidence.
func readTrialCounts(r *blockReader) (*big.Float, *big.Float) {
	trials, trialsParsed := field(r, "trials", true, numberOf)
	trialsValid := trialsParsed && trials.IsInt() && trials.Sign() > 0
	if trialsParsed && !trialsValid {
		r.failCoded(codeInvalidEvidence, attributeRange(r, "trials"), "trials must be a positive whole number")
	}

	failures, failuresParsed := field(r, "failures", true, numberOf)
	if failuresParsed {
		switch {
		case !failures.IsInt() || failures.Sign() < 0:
			r.failCoded(codeInvalidEvidence, attributeRange(r, "failures"), "failures must be a whole number from zero up")
		case trialsValid && failures.Cmp(trials) > 0:
			r.failCoded(codeInvalidEvidence, attributeRange(r, "failures"), "failures cannot exceed trials")
		}
	}
	return trials, failures
}

// decodePV decodes one physical variable block.
func decodePV(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindPV), aliasedMeta)
	realizes := r.requiredRefs("realizes")
	kind := r.optionalString("kind")
	address := r.optionalString("address")
	note := r.optionalString("note")

	core.PVs = append(core.PVs, model.PV{
		Header:   header,
		Realizes: realizes,
		Kind:     kind,
		Address:  address,
		Note:     note,
	})
}

// decodeModule decodes one module block. A module realizes design properties
// or contains other modules, never both.
func decodeModule(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindModule), aliasedMeta)
	if attributePresent(r, "realizes") && attributePresent(r, "contains") {
		r.fail(blockHeaderRange(r.block), "a module cannot have both realizes and contains")
	}
	realizes := optionalList(r, "realizes", nonEmptyRefsOf)
	contains := optionalList(r, "contains", nonEmptyRefsOf)
	within := r.optionalRef("within")
	inputs := optionalList(r, "inputs", stringsOf)
	outputs := optionalList(r, "outputs", stringsOf)
	ownsState := optionalList(r, "owns_state", refsOf)
	dependencies := optionalList(r, "dependencies", refsExcluding(header.ID))
	invariants := optionalList(r, "invariants", stringsOf)
	acceptanceChecks := optionalList(r, "acceptance_checks", refsOf)
	failureBehavior := r.optionalString("failure_behavior")

	core.Modules = append(core.Modules, model.Module{
		Header:           header,
		Realizes:         realizes,
		Contains:         contains,
		Within:           within,
		Inputs:           inputs,
		Outputs:          outputs,
		OwnsState:        ownsState,
		Dependencies:     dependencies,
		Invariants:       invariants,
		FailureBehavior:  failureBehavior,
		AcceptanceChecks: acceptanceChecks,
	})
}

// decodeFlow decodes one flow block and its unlabeled loop blocks.
func decodeFlow(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindFlow), standardMeta)
	steps, _ := field(r, "steps", true, stepsOf)
	owner := r.optionalRef("owner")
	input := r.optionalString("input")
	output := r.optionalString("output")
	after, _ := field(r, "after", false, refsOf)

	var loops []model.Loop
	for _, block := range r.childBlocks("loop") {
		loops = append(loops, decodeLoop(r, block))
	}

	core.Flows = append(core.Flows, model.Flow{
		Header: header,
		Steps:  steps,
		Owner:  owner,
		Input:  input,
		Output: output,
		After:  after,
		Loops:  loops,
	})
}

// stepsOf parses the steps of a flow. Each step is one reference or a
// nonempty flat list of references, and each identity appears at most once.
func stepsOf(expr hclsyntax.Expression) ([][]model.Ref, string) {
	items, ok := listItems(expr)
	if !ok {
		return nil, "must be a list of steps"
	}
	if len(items) == 0 {
		return nil, "must list at least one step"
	}
	steps := make([][]model.Ref, 0, len(items))
	seen := make(map[model.ID]bool)
	for _, item := range items {
		group, problem := stepGroupOf(item)
		if problem != "" {
			return nil, problem
		}
		for _, ref := range group {
			if seen[ref.Target] {
				return nil, fmt.Sprintf("%s appears more than once in the flow", ref.Target)
			}
			seen[ref.Target] = true
		}
		steps = append(steps, group)
	}
	return steps, ""
}

// stepGroupOf parses one step: a single reference or a nonempty flat list.
func stepGroupOf(expr hclsyntax.Expression) ([]model.Ref, string) {
	if _, isGroup := expr.(*hclsyntax.TupleConsExpr); isGroup {
		return nonEmptyRefsOf(expr)
	}
	ref, problem := refOf(expr)
	if problem != "" {
		return nil, problem
	}
	return []model.Ref{ref}, ""
}

// decodeLoop decodes one unlabeled loop block nested in a flow.
func decodeLoop(parent *blockReader, block *hclsyntax.Block) model.Loop {
	r := newBlockReader(parent.file, block, parent.findings)
	header := r.unlabeledHeader(standardMeta)
	after := r.requiredRef("after")
	backTo := r.requiredRef("back_to")
	when := r.requiredString("when")
	bound := r.requiredInteger("bound", true)
	r.finish()

	return model.Loop{
		Unlabeled: header,
		After:     after,
		BackTo:    backTo,
		When:      when,
		Bound:     bound,
	}
}

// decodeScenario decodes one scenario block.
func decodeScenario(r *blockReader, core *model.Core) {
	header := r.namedHeader(string(model.KindScenario), standardMeta)
	kind := model.ScenarioKind(r.requiredEnum("kind", string(model.ScenarioChange), string(model.ScenarioOperational), string(model.ScenarioFailure)))
	verifies := optionalList(r, "verifies", refsOf)
	vary := r.optionalRefs("vary")
	expectAffected := r.optionalRefs("expect_affected")
	expectUnaffected := r.optionalRefs("expect_unaffected")
	steps := optionalList(r, "steps", stringsOf)
	description := r.optionalString("description")
	under := r.optionalRef("under")

	if shared, ok := sharedRef(expectAffected, expectUnaffected); ok {
		r.fail(attributeRange(r, "expect_unaffected"), fmt.Sprintf("%s cannot be both affected and unaffected", shared))
	}

	core.Scenarios = append(core.Scenarios, model.Scenario{
		Header:           header,
		Kind:             kind,
		Verifies:         verifies,
		Vary:             vary,
		ExpectAffected:   expectAffected,
		ExpectUnaffected: expectUnaffected,
		Steps:            steps,
		Description:      description,
		Under:            under,
	})
}

// sharedRef returns the first identity that appears in both lists.
func sharedRef(first, second []model.Ref) (model.ID, bool) {
	seen := make(map[model.ID]bool, len(first))
	for _, ref := range first {
		seen[ref.Target] = true
	}
	for _, ref := range second {
		if seen[ref.Target] {
			return ref.Target, true
		}
	}
	return "", false
}
