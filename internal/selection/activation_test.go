package selection

import (
	"fmt"
	"slices"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/schema"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

// activationDesign is the design root, its need, and the root FR that every fixture shares.
const activationDesign = `design "checkout" {
  boundary = "Record each payment that the checkout accepts."
}

need "record_payments" {
  statement = "Record every accepted payment."
}

fr "store" {
  need      = [need.record_payments]
  statement = "Store each accepted payment record."
}
`

// activationAudit is a selected DP that no candidate lists, so it is a common DP.
const activationAudit = `dp "audit_trail" {
  controls       = [fr.store]
  responsibility = "Write each stored payment to the audit log."
  decision       = "selected"
}
`

// activationReportFR is a root FR that no DP controls.
const activationReportFR = `fr "report" {
  need      = [need.record_payments]
  statement = "Report the total of accepted payments."
}
`

// activationCacheTree is a child FR under a proposed engine DP, with a proposed cache DP
// under it. The fast candidate lists only the cache DP. The layered candidate lists both.
const activationCacheTree = `dp "engine" {
  controls       = [fr.store]
  responsibility = "Choose the storage engine."
  decision       = "proposed"
}

fr "cache_hit" {
  parent    = dp.engine
  statement = "Serve repeated reads from memory."
}

dp "cache" {
  controls       = [fr.cache_hit]
  responsibility = "Hold recent records in memory."
  decision       = "proposed"
}

decision "cache_layer" {
  addresses = [fr.cache_hit]

  candidate "fast" {
    dps = [dp.cache]
  }

  candidate "layered" {
    dps = [dp.engine, dp.cache]
  }
}
`

// activationMultiSourceInfluence has two source DPs, both of which can be active.
const activationMultiSourceInfluence = `influence {
  from      = [dp.card_path, dp.audit_trail]
  to        = fr.store
  effect    = "material"
  mechanism = "Changing either writer changes the stored record."
}
`

// activationCardAndWalletInfluence has two source DPs, and only the card DP is active by default.
const activationCardAndWalletInfluence = `influence {
  from      = [dp.card_path, dp.wallet_path]
  to        = fr.store
  effect    = "material"
  mechanism = "Changing either writer changes the stored record."
}
`

// activationCardToCacheInfluence targets a child FR under an inactive engine DP.
const activationCardToCacheInfluence = `influence {
  from      = [dp.card_path]
  to        = fr.cache_hit
  effect    = "material"
  mechanism = "Changing the card writer changes the cached record."
}
`

// activationPaymentDecision is a decision with a card candidate and a wallet candidate. Each DP
// takes the lifecycle state that the caller supplies.
func activationPaymentDecision(cardState, walletState string) string {
	return fmt.Sprintf(`dp "card_path" {
  controls       = [fr.store]
  responsibility = "Store card payment records."
  decision       = %q
}

dp "wallet_path" {
  controls       = [fr.store]
  responsibility = "Store wallet payment records."
  decision       = %q
}

decision "payment" {
  addresses = [fr.store]
  selected  = "card"

  candidate "card" {
    dps = [dp.card_path]
  }

  candidate "wallet" {
    dps = [dp.wallet_path]
  }
}
`, cardState, walletState)
}

// activationLegacyDecision is a decision with one candidate that lists a DP in the given state.
func activationLegacyDecision(lifecycle string) string {
	return fmt.Sprintf(`dp "legacy_path" {
  controls       = [fr.store]
  responsibility = "Store records in the retired file format."
  decision       = %q
}

decision "retired" {
  addresses = [fr.store]

  candidate "legacy" {
    dps = [dp.legacy_path]
  }
}
`, lifecycle)
}

// buildGraph parses each inline source as one core file, decodes the files, and resolves them.
// It fails the test when any of those setup steps reports a finding.
func buildGraph(t *testing.T, sources ...string) *model.Graph {
	t.Helper()
	units := make([]syntax.Unit, 0, len(sources))
	for index, source := range sources {
		unit := syntax.Parse(model.Source{Path: fmt.Sprintf("activation-%d.ad.hcl", index+1), Bytes: []byte(source)})
		if len(unit.Findings) != 0 {
			t.Fatalf("source %d has %d native parse findings, want none", index+1, len(unit.Findings))
		}
		units = append(units, unit)
	}
	core, decodeFindings := schema.DecodeCore(units)
	if len(decodeFindings) != 0 {
		t.Fatalf("decode produced %d findings, want none", len(decodeFindings))
	}
	graph, resolveFindings := schema.Resolve(core)
	if len(resolveFindings) != 0 {
		t.Fatalf("resolve produced %d findings, want none", len(resolveFindings))
	}
	if graph == nil {
		t.Fatal("resolve returned no graph, want a graph")
	}
	return graph
}

// activationRequireValid fails unless the selection returned a snapshot and no findings.
func activationRequireValid(t *testing.T, snapshot *Snapshot, findings []result.Finding) *Snapshot {
	t.Helper()
	if len(findings) != 0 {
		t.Fatalf("selection produced %d findings, want none", len(findings))
	}
	if snapshot == nil {
		t.Fatal("snapshot is nil, want a snapshot")
	}
	return snapshot
}

// activationRequireInvalid fails unless the selection returned no snapshot and only
// invalid_selection findings.
func activationRequireInvalid(t *testing.T, snapshot *Snapshot, findings []result.Finding) {
	t.Helper()
	if snapshot != nil {
		t.Fatal("snapshot is non-nil, want nil for an invalid selection")
	}
	if len(findings) == 0 {
		t.Fatal("selection produced no findings, want invalid_selection")
	}
	for _, finding := range findings {
		if finding.Code != "invalid_selection" {
			t.Fatalf("finding code = %q, want invalid_selection", finding.Code)
		}
	}
}

// activationRequireActive fails unless exactly the wanted IDs are true in members.
func activationRequireActive(t *testing.T, label string, members map[model.ID]bool, want ...model.ID) {
	t.Helper()
	var active []model.ID
	for id, on := range members {
		if on {
			active = append(active, id)
		}
	}
	activationRequireIDs(t, "active "+label, active, want...)
}

// activationRequireIDs fails unless got holds exactly the wanted IDs, in any order.
func activationRequireIDs(t *testing.T, label string, got []model.ID, want ...model.ID) {
	t.Helper()
	gotText := activationText(got)
	wantText := activationText(want)
	if !slices.Equal(gotText, wantText) {
		t.Fatalf("%s = %v, want %v", label, gotText, wantText)
	}
}

// activationText returns the IDs as sorted strings.
func activationText(ids []model.ID) []string {
	text := make([]string, 0, len(ids))
	for _, id := range ids {
		text = append(text, string(id))
	}
	slices.Sort(text)
	return text
}

// activationSources lists the source DP of each applied influence.
func activationSources(applied []AppliedInfluence) []model.ID {
	sources := make([]model.ID, 0, len(applied))
	for _, influence := range applied {
		sources = append(sources, influence.From)
	}
	return sources
}

// activationTargets lists the identities that a list of references names.
func activationTargets(refs []model.Ref) []model.ID {
	targets := make([]model.ID, 0, len(refs))
	for _, ref := range refs {
		targets = append(targets, ref.Target)
	}
	return targets
}

func TestSelectDefaultActivatesSelectedDPsOnly(t *testing.T) {
	// Given a payment decision whose card DP is selected, whose wallet DP is proposed, and a selected common audit DP.
	graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"), activationAudit)

	// When the default snapshot is selected.
	snapshot, findings := Select(graph, Request{})

	// Then the snapshot has no candidate, the selected DPs are active, and only the root FR is active.
	snapshot = activationRequireValid(t, snapshot, findings)
	if snapshot.Candidate != "" {
		t.Fatalf("candidate = %q, want empty for the default snapshot", snapshot.Candidate)
	}
	activationRequireActive(t, "DPs", snapshot.ActiveDPs, "dp.card_path", "dp.audit_trail")
	activationRequireActive(t, "FRs", snapshot.ActiveFRs, "fr.store")
}

func TestSelectDefaultLeavesUnselectedAlternativeInGraph(t *testing.T) {
	// Given a payment decision whose card DP is selected and whose wallet DP is the unselected alternative.
	graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"))

	// When the default snapshot is selected.
	snapshot, findings := Select(graph, Request{})

	// Then the wallet DP is absent from the active set and is still declared in the graph.
	snapshot = activationRequireValid(t, snapshot, findings)
	activationRequireActive(t, "DPs", snapshot.ActiveDPs, "dp.card_path")
	if !graph.Has("dp.wallet_path") {
		t.Fatal("graph does not declare dp.wallet_path, want the unselected alternative kept")
	}
}

func TestSelectRejectsDPDeclaredByDistinctDecisions(t *testing.T) {
	overlap := `decision "duplicate" {
  addresses = [fr.store]
  selected  = "shared"

  candidate "shared" {
    dps = [dp.card_path]
  }
}
`
	graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"), overlap)
	for _, candidate := range []model.ID{"", "decision.payment.candidate.card", "decision.payment.candidate.wallet", "decision.duplicate.candidate.shared"} {
		t.Run(string(candidate), func(t *testing.T) {
			snapshot, findings := Select(graph, Request{Candidate: candidate})
			activationRequireInvalid(t, snapshot, findings)
			if len(findings) != 1 {
				t.Fatalf("findings = %d, want one overlap finding", len(findings))
			}
			if findings[0].Subject != "dp.card_path" || findings[0].Range == nil {
				t.Fatalf("overlap finding = %#v, want the shared DP and its authored range", findings[0])
			}
		})
	}
}

func TestSelectAcceptsDisjointDecisions(t *testing.T) {
	disjoint := `decision "audit" {
  addresses = [fr.store]
  selected  = "log"

  candidate "log" {
    dps = [dp.audit_trail]
  }
}
`
	graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"), activationAudit, disjoint)
	cases := []struct {
		candidate model.ID
		active    []model.ID
	}{
		{candidate: "", active: []model.ID{"dp.card_path", "dp.audit_trail"}},
		{candidate: "decision.payment.candidate.wallet", active: []model.ID{"dp.wallet_path"}},
		{candidate: "decision.audit.candidate.log", active: []model.ID{"dp.audit_trail"}},
	}
	for _, tc := range cases {
		t.Run(string(tc.candidate), func(t *testing.T) {
			snapshot, findings := Select(graph, Request{Candidate: tc.candidate})
			snapshot = activationRequireValid(t, snapshot, findings)
			activationRequireActive(t, "DPs", snapshot.ActiveDPs, tc.active...)
		})
	}
}

func TestSelectAcceptsSharedDPInAlternativesOfOneDecision(t *testing.T) {
	alternatives := `decision "audit" {
  addresses = [fr.store]
  selected  = "first"

  candidate "first" {
    dps = [dp.audit_trail]
  }

  candidate "second" {
    dps = [dp.audit_trail]
  }
}
`
	graph := buildGraph(t, activationDesign, activationAudit, alternatives)
	for _, candidate := range []model.ID{"", "decision.audit.candidate.first", "decision.audit.candidate.second"} {
		t.Run(string(candidate), func(t *testing.T) {
			snapshot, findings := Select(graph, Request{Candidate: candidate})
			snapshot = activationRequireValid(t, snapshot, findings)
			activationRequireActive(t, "DPs", snapshot.ActiveDPs, "dp.audit_trail")
		})
	}
}

func TestSelectCompetingCandidatesActivateDifferentDPs(t *testing.T) {
	// Given a payment decision with a selected card candidate and a proposed wallet candidate.
	graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"))

	// When the default snapshot and the wallet candidate are each selected.
	defaultSnapshot, defaultFindings := Select(graph, Request{})
	walletSnapshot, walletFindings := Select(graph, Request{Candidate: "decision.payment.candidate.wallet"})

	// Then each snapshot activates only its own DP.
	defaultSnapshot = activationRequireValid(t, defaultSnapshot, defaultFindings)
	walletSnapshot = activationRequireValid(t, walletSnapshot, walletFindings)
	activationRequireActive(t, "default DPs", defaultSnapshot.ActiveDPs, "dp.card_path")
	activationRequireActive(t, "wallet DPs", walletSnapshot.ActiveDPs, "dp.wallet_path")
}

func TestSelectExplicitCandidateActivatesItsDPsAndCommonSelectedDPs(t *testing.T) {
	// Given a payment decision with a proposed wallet candidate and a selected DP that no candidate lists.
	graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"), activationAudit)

	// When the wallet candidate is selected.
	snapshot, findings := Select(graph, Request{Candidate: "decision.payment.candidate.wallet"})

	// Then the snapshot names that candidate and activates exactly its wallet DP and the common audit DP.
	snapshot = activationRequireValid(t, snapshot, findings)
	if snapshot.Candidate != "decision.payment.candidate.wallet" {
		t.Fatalf("candidate = %q, want decision.payment.candidate.wallet", snapshot.Candidate)
	}
	activationRequireActive(t, "DPs", snapshot.ActiveDPs, "dp.wallet_path", "dp.audit_trail")
}

func TestSelectExplicitCandidateExcludesDPListedByAnotherCandidate(t *testing.T) {
	// Given a payment decision whose selected card DP is listed only by the card candidate.
	graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"))

	// When the wallet candidate is selected.
	snapshot, findings := Select(graph, Request{Candidate: "decision.payment.candidate.wallet"})

	// Then the card DP stays inactive, because another candidate lists it, even though its lifecycle is selected.
	snapshot = activationRequireValid(t, snapshot, findings)
	activationRequireActive(t, "DPs", snapshot.ActiveDPs, "dp.wallet_path")
}

func TestSelectExplicitCandidateAcceptsProposedDP(t *testing.T) {
	// Given a payment decision whose card and wallet DPs are both proposed.
	graph := buildGraph(t, activationDesign, activationPaymentDecision("proposed", "proposed"))

	// When the card candidate, which lists a proposed DP, is selected.
	snapshot, findings := Select(graph, Request{Candidate: "decision.payment.candidate.card"})

	// Then the proposed DP is active and no finding is reported.
	snapshot = activationRequireValid(t, snapshot, findings)
	activationRequireActive(t, "DPs", snapshot.ActiveDPs, "dp.card_path")
}

func TestSelectRejectsCandidateListingRetiredDP(t *testing.T) {
	cases := []struct {
		name      string
		lifecycle string
	}{
		{name: "rejected", lifecycle: "rejected"},
		{name: "superseded", lifecycle: "superseded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a payment decision and a retired decision whose legacy candidate lists a DP in the given lifecycle state.
			graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"), activationLegacyDecision(tc.lifecycle))

			// When the legacy candidate is selected.
			snapshot, findings := Select(graph, Request{Candidate: "decision.retired.candidate.legacy"})

			// Then the selection is invalid and returns no snapshot.
			activationRequireInvalid(t, snapshot, findings)
		})
	}
}

func TestSelectDefaultRejectsSelectedLabelThatDisagreesWithLifecycle(t *testing.T) {
	cases := []struct {
		name        string
		cardState   string
		walletState string
	}{
		{name: "selected candidate DP is proposed", cardState: "proposed", walletState: "proposed"},
		{name: "other candidate DP is selected", cardState: "selected", walletState: "selected"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a payment decision whose selected label is card and whose DP lifecycle states disagree with that label.
			graph := buildGraph(t, activationDesign, activationPaymentDecision(tc.cardState, tc.walletState))

			// When the default snapshot is selected.
			snapshot, findings := Select(graph, Request{})

			// Then the selection is invalid and returns no snapshot.
			activationRequireInvalid(t, snapshot, findings)
		})
	}
}

func TestSelectRejectsUnknownCandidate(t *testing.T) {
	cases := []struct {
		name      string
		candidate model.ID
	}{
		{name: "unknown candidate label", candidate: "decision.payment.candidate.missing"},
		{name: "unknown decision", candidate: "decision.nothing.candidate.card"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a payment decision with card and wallet candidates.
			graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"))

			// When an undeclared candidate ID is selected.
			snapshot, findings := Select(graph, Request{Candidate: tc.candidate})

			// Then the selection is invalid and returns no snapshot.
			activationRequireInvalid(t, snapshot, findings)
		})
	}
}

func TestSelectRejectsCandidateWhoseDPHasInactiveParent(t *testing.T) {
	// Given a cache tree whose parent engine DP is proposed, and a fast candidate that lists only the child cache DP.
	graph := buildGraph(t, activationDesign, activationCacheTree)

	// When the fast candidate is selected.
	snapshot, findings := Select(graph, Request{Candidate: "decision.cache_layer.candidate.fast"})

	// Then the selection is invalid and returns no snapshot, so the missing parent DP is not inserted.
	activationRequireInvalid(t, snapshot, findings)
}

func TestSelectActivatesParentWhenCandidateListsIt(t *testing.T) {
	// Given a cache tree whose layered candidate lists both the parent engine DP and the child cache DP.
	graph := buildGraph(t, activationDesign, activationCacheTree)

	// When the layered candidate is selected.
	snapshot, findings := Select(graph, Request{Candidate: "decision.cache_layer.candidate.layered"})

	// Then both listed DPs and the FRs they control are active, with no finding.
	snapshot = activationRequireValid(t, snapshot, findings)
	activationRequireActive(t, "DPs", snapshot.ActiveDPs, "dp.engine", "dp.cache")
	activationRequireActive(t, "FRs", snapshot.ActiveFRs, "fr.store", "fr.cache_hit")
}

func TestSelectDefaultIgnoresChildFRsOfInactiveDP(t *testing.T) {
	// Given a payment decision with a selected card DP and a cache tree whose engine and cache DPs are proposed.
	graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"), activationCacheTree)

	// When the default snapshot is selected.
	snapshot, findings := Select(graph, Request{})

	// Then the child FR under the inactive engine DP is inactive and is not reported missing.
	snapshot = activationRequireValid(t, snapshot, findings)
	activationRequireActive(t, "DPs", snapshot.ActiveDPs, "dp.card_path")
	activationRequireActive(t, "FRs", snapshot.ActiveFRs, "fr.store")
}

func TestSelectDefaultKeepsRootFRsActive(t *testing.T) {
	// Given a root FR that no DP controls, and a payment decision with a selected card DP.
	graph := buildGraph(t, activationDesign, activationReportFR, activationPaymentDecision("selected", "proposed"))

	// When the default snapshot is selected.
	snapshot, findings := Select(graph, Request{})

	// Then both root FRs are active, including the one that no DP controls.
	snapshot = activationRequireValid(t, snapshot, findings)
	activationRequireActive(t, "FRs", snapshot.ActiveFRs, "fr.store", "fr.report")
}

func TestSelectModuleKeepsOnlyActiveRealizedDPs(t *testing.T) {
	// Given a module that realizes the selected card DP and the proposed wallet DP.
	graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"), `module "gateway" {
  realizes = [dp.card_path, dp.wallet_path]
}
`)

	// When the default snapshot is selected.
	snapshot, findings := Select(graph, Request{})

	// Then the gateway module is active and keeps only its active realized DP.
	snapshot = activationRequireValid(t, snapshot, findings)
	if len(snapshot.Modules) != 1 {
		t.Fatalf("active modules = %d, want 1", len(snapshot.Modules))
	}
	activationRequireIDs(t, "gateway realized DPs", snapshot.Modules["module.gateway"], "dp.card_path")
}

func TestSelectModuleWithNoActiveDPIsInactive(t *testing.T) {
	// Given a module whose only realized DP is proposed.
	graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"), activationLegacyDecision("proposed"), `module "legacy_service" {
  realizes = [dp.legacy_path]
}
`)

	// When the default snapshot is selected.
	snapshot, findings := Select(graph, Request{})

	// Then the module is not active, so the snapshot has no module entries.
	snapshot = activationRequireValid(t, snapshot, findings)
	if _, present := snapshot.Modules["module.legacy_service"]; present {
		t.Fatal("module.legacy_service is active, want inactive because its only realized DP is proposed")
	}
	if len(snapshot.Modules) != 0 {
		t.Fatalf("active modules = %d, want 0", len(snapshot.Modules))
	}
}

func TestSelectExpandsMultiSourceInfluenceIntoOneRecordPerSource(t *testing.T) {
	// Given an influence with two source DPs, both of which are active by default.
	graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"), activationAudit, activationMultiSourceInfluence)

	// When the default snapshot is selected.
	snapshot, findings := Select(graph, Request{})

	// Then the influence becomes two applied entries, one per source DP, and each keeps the authored record.
	snapshot = activationRequireValid(t, snapshot, findings)
	if len(snapshot.Influences) != 2 {
		t.Fatalf("applied influences = %d, want 2", len(snapshot.Influences))
	}
	activationRequireIDs(t, "influence sources", activationSources(snapshot.Influences), "dp.card_path", "dp.audit_trail")
	for _, applied := range snapshot.Influences {
		if applied.To != "fr.store" {
			t.Fatalf("applied influence target = %q, want fr.store", applied.To)
		}
		if applied.Record == nil {
			t.Fatal("applied influence has no record, want the authored record")
		}
		activationRequireIDs(t, "authored record sources", activationTargets(applied.Record.From), "dp.card_path", "dp.audit_trail")
	}
}

func TestSelectDropsInfluenceWithInactiveSource(t *testing.T) {
	// Given an influence whose sources are the selected card DP and the proposed wallet DP.
	graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"), activationCardAndWalletInfluence)

	// When the default snapshot is selected.
	snapshot, findings := Select(graph, Request{})

	// Then only the card source remains as an applied influence, and the authored record still lists both sources.
	snapshot = activationRequireValid(t, snapshot, findings)
	activationRequireIDs(t, "applied influence sources", activationSources(snapshot.Influences), "dp.card_path")
	if len(graph.Core.Influences) != 1 {
		t.Fatalf("authored influences = %d, want 1", len(graph.Core.Influences))
	}
	activationRequireIDs(t, "authored record sources", activationTargets(graph.Core.Influences[0].From), "dp.card_path", "dp.wallet_path")
}

func TestSelectDropsInfluenceWithInactiveTarget(t *testing.T) {
	// Given an influence from the selected card DP to a child FR under an inactive engine DP.
	graph := buildGraph(t, activationDesign, activationPaymentDecision("selected", "proposed"), activationCacheTree, activationCardToCacheInfluence)

	// When the default snapshot is selected.
	snapshot, findings := Select(graph, Request{})

	// Then the influence is dropped from the snapshot, the target FR is inactive, and the authored record stays in the graph.
	snapshot = activationRequireValid(t, snapshot, findings)
	if len(snapshot.Influences) != 0 {
		t.Fatalf("applied influences = %d, want 0 when the target FR is inactive", len(snapshot.Influences))
	}
	activationRequireActive(t, "FRs", snapshot.ActiveFRs, "fr.store")
	if len(graph.Core.Influences) != 1 {
		t.Fatalf("authored influences = %d, want 1", len(graph.Core.Influences))
	}
}
