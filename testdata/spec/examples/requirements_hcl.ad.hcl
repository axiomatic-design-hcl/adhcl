# A proposed design for RequirementsHCL and its analyzer.
# Selected means a chosen design mechanism, not implemented or verified code.
# The review supplies baseline evidence, not proof of the proposed matrix.
# The semantic_changes context describes the conditional semantic model.
# Unvalidated premises keep its support unresolved.
# The process_failure context retains selected shared-resource unknowns.
# No binding probe, observation result, or probability score exists here.
# This models the core assessment pipeline and a separate extension subtree.
# Their matrices do not establish whole-system independence.
# The planned Go tool and attachment adapters are not implemented.

source "review_2026_10_28" {
  title = "Review of RequirementsHCL and its prototype"
  url = "../docs/review-2026-10-28.md"
}

design "requirements_hcl" {
  boundary = "Author, resolve, assess, and report AD designs. Binding observation remains a target contract. Deployment and system enforcement are outside this boundary."
  level = "architecture"
  workload = workload.semantic_changes
  cites = source.review_2026_10_28
}

workload "semantic_changes" {
  description = "Incomplete designs, malformed input, alternative candidates, and named conditions within fixed semantic contracts. No latency or availability bound is supplied."
}

workload "process_failure" {
  description = "The analyzer shares a process, memory, CPU time, and file access. Failure effects have no measured isolation bounds."
}

context "semantic_changes" {
  description = "Assess semantic changes within the stated contracts."
  workload = workload.semantic_changes
  excludes = [context.process_failure]
}

context "process_failure" {
  description = "Assess shared-resource failures without an isolation claim."
  workload = workload.process_failure
  excludes = [context.semantic_changes]
}

need "faithful_design" {
  statement = "Preserve the meaning of an AD design."
  why = "Reviewers must distinguish facts, proposed controls, and supported conclusions."
  origin = "inferred"
}

need "safe_input" {
  statement = "Reject unsafe or ambiguous input without losing accepted facts."
  why = "People and AI agents need diagnostics instead of silent replacement."
  origin = "inferred"
}

need "traceable_change" {
  statement = "Keep design changes and implementation evidence traceable."
  why = "A selected design does not establish its implementation."
  origin = "inferred"
}

need "tool_exchange" {
  statement = "Exchange design information with separate behavior and infrastructure tools."
  why = "Each model must retain its own meaning and ownership."
  origin = "inferred"
}

need "readable_changes" {
  statement = "Make source changes and derived reports readable."
  why = "Reviewers must locate changed facts and their effects."
  origin = "inferred"
}

constraint "native_hcl" {
  kind = "technology"
  statement = "Use Go github.com/hashicorp/hcl/v2 for native HCL syntax. Admit literal values and static traversals only, without computed expressions, templates, or functions. Do not inherit the Python parser's numeric or heredoc limits."
  origin = "inferred"
}

constraint "separate_languages" {
  kind = "compatibility"
  statement = "Keep AD and external behavior models separate. Put tool-specific realization and behavior links in separately owned attachments, outside core facts."
  origin = "inferred"
}

assumption "semantic_contracts" {
  statement = "Within one schema version, consumers cannot mutate upstream facts or redefine upstream policy. Binding assessment owns both observation and comparison."
  status = "to_validate"
}

assumption "fixed_expression_profile" {
  statement = "AD schema changes do not change the admitted syntax or expression profile. A profile change requires a new syntax decision."
  status = "to_validate"
}

fr "qualified_assessment" {
  alias = "FR0"
  need = [need.faithful_design, need.safe_input, need.traceable_change, need.tool_exchange, need.readable_changes]
  statement = "Produce a traceable AD assessment for a named design and condition."
  criterion "assessment" {
    metric = "All in-scope findings retain identity, scope, conditions, and evidence status."
    op = "=="
    value = true
    under = workload.semantic_changes
    origin = "proposed"
  }
}

dp "assessment_pipeline" {
  alias = "DP0"
  controls = [fr.qualified_assessment]
  responsibility = "Separate syntax admission, resolution, selection, binding assessment, analysis, and reporting through explicit contracts."
  decision = "selected"
  bounded_by = [constraint.native_hcl, constraint.separate_languages]
}

fr "capture" {
  alias = "FR1"
  parent = dp.assessment_pipeline
  need = [need.faithful_design, need.safe_input, need.readable_changes]
  statement = "Preserve accepted design facts and reject invalid structure before dependent analysis."
  criterion "lossless" {
    metric = "Accepted entities, values, typed references, and source positions retain identity without silent replacement."
    op = "=="
    value = true
    allocated_from = fr.qualified_assessment.criterion.assessment
    origin = "proposed"
  }
}

fr "trace_selection" {
  alias = "FR5"
  parent = dp.assessment_pipeline
  need = [need.faithful_design, need.traceable_change]
  statement = "Retain candidate identity, provenance, and applicable premises for every analyzed relation."
  criterion "traceable" {
    metric = "A candidate change preserves alternatives and withdraws dependent results from the previous snapshot."
    op = "=="
    value = true
    allocated_from = fr.qualified_assessment.criterion.assessment
    origin = "proposed"
  }
}

fr "detect_drift" {
  alias = "FR3"
  parent = dp.assessment_pipeline
  need = [need.traceable_change, need.tool_exchange]
  statement = "Distinguish observed binding drift from unavailable implementation evidence."
  criterion "comparison" {
    metric = "A seeded binding mismatch yields drift, while unavailable observations yield unknown rather than verified."
    op = "=="
    value = true
    allocated_from = fr.qualified_assessment.criterion.assessment
    origin = "proposed"
  }
}

fr "assess_method" {
  alias = "FR2"
  parent = dp.assessment_pipeline
  need = [need.faithful_design, need.safe_input]
  statement = "Derive qualified AD conclusions without treating missing facts as independence."
  criterion "qualified" {
    metric = "Exact effects, conflicts, assumptions, unknown cells, and refinement obligations remain visible in every applicable conclusion."
    op = "=="
    value = true
    allocated_from = fr.qualified_assessment.criterion.assessment
    origin = "proposed"
  }
}

fr "present" {
  alias = "FR4"
  parent = dp.assessment_pipeline
  need = [need.tool_exchange, need.readable_changes]
  statement = "Present consistent assessment results and reliable command outcomes."
  criterion "faithful_reports" {
    metric = "Every report preserves common cell values and conditions. Invalid input and failed assertions produce nonzero status."
    op = "=="
    value = true
    allocated_from = fr.qualified_assessment.criterion.assessment
    origin = "proposed"
  }
}

dp "checked_graph" {
  alias = "DP1"
  controls = [fr.capture]
  responsibility = "Use a versioned syntax profile, schema decoder, and typed resolver to retain each accepted fact."
  decision = "selected"
}

dp "candidate_snapshot" {
  alias = "DP5"
  controls = [fr.trace_selection]
  responsibility = "Select a coherent candidate and context without deleting other records or overwriting matching relations."
  decision = "selected"
  owns_state = [state.snapshot]
}

dp "binding_assessment" {
  alias = "DP3"
  controls = [fr.detect_drift]
  responsibility = "Own observations, comparison, and provenance for declared bindings. This target mechanism has no implemented adapter."
  decision = "selected"
  owns_state = [state.drift_facts]
}

dp "qualified_analysis" {
  alias = "DP2"
  controls = [fr.assess_method]
  responsibility = "Resolve one qualified influence model for consistency, matrices, partial order, cross-branch effects, and impact."
  decision = "selected"
  owns_state = [state.assessment]
}

dp "read_only_reports" {
  alias = "DP4"
  controls = [fr.present]
  responsibility = "Render a common assessment without changing its facts, conditions, or error status."
  decision = "selected"
  owns_state = [state.report_output]
}

# The pairing identifies a proposed control. Its mechanism is not success evidence.
influence {
  from = [dp.assessment_pipeline]
  to = fr.qualified_assessment
  effect = "material"
  mechanism = "Stage contracts retain each fact and its conditions through the assessment. A broken contract can invalidate the parent criterion."
  mechanism_source = "inferred"
}

influence {
  from = [dp.checked_graph]
  to = fr.capture
  effect = "material"
  mechanism = "The decoder and resolver decide which identities, values, and references survive admission."
  mechanism_source = "inferred"
  under = workload.semantic_changes
}

influence {
  from = [dp.candidate_snapshot]
  to = fr.trace_selection
  effect = "material"
  mechanism = "Selection policy determines the active facts and the provenance attached to each relation."
  mechanism_source = "inferred"
  under = workload.semantic_changes
}

influence {
  from = [dp.binding_assessment]
  to = fr.detect_drift
  effect = "material"
  mechanism = "The comparison contract distinguishes a mismatch from unavailable observations. Its implementation still needs evidence."
  mechanism_source = "inferred"
  under = workload.semantic_changes
}

influence {
  from = [dp.qualified_analysis]
  to = fr.assess_method
  effect = "material"
  mechanism = "Cell resolution and rule policy determine whether conclusions preserve unknowns, conflicts, and premises."
  mechanism_source = "inferred"
  under = workload.semantic_changes
}

influence {
  from = [dp.read_only_reports]
  to = fr.present
  effect = "material"
  mechanism = "Report projection and exit policy determine whether callers receive complete findings and failure status."
  mechanism_source = "inferred"
  under = workload.semantic_changes
}

# Aggregate effects require later allocation to leaves. They are not leaf facts.
influence {
  from = [dp.checked_graph]
  to = fr.trace_selection
  effect = "material"
  mechanism = "Lost identity or provenance prevents the snapshot from tracing a relation to its original record."
  mechanism_source = "inferred"
  under = workload.semantic_changes
}

influence {
  from = [dp.checked_graph, dp.candidate_snapshot]
  to = fr.detect_drift
  effect = "material"
  mechanism = "The graph and selected candidate determine the declared bindings that an adapter must compare."
  mechanism_source = "inferred"
  under = workload.semantic_changes
}

influence {
  from = [dp.checked_graph, dp.candidate_snapshot, dp.binding_assessment]
  to = fr.assess_method
  effect = "material"
  mechanism = "Accepted facts, candidate selection, and drift facts change which method and realization findings the assessment must preserve."
  mechanism_source = "inferred"
  under = workload.semantic_changes
}

influence {
  from = [dp.checked_graph, dp.candidate_snapshot, dp.binding_assessment, dp.qualified_analysis]
  to = fr.present
  effect = "material"
  mechanism = "Changes to source identity, selection, drift facts, or conclusions change the findings that a faithful report must expose."
  mechanism_source = "inferred"
  under = workload.semantic_changes
}

influence {
  from = [dp.candidate_snapshot, dp.binding_assessment, dp.qualified_analysis, dp.read_only_reports]
  to = fr.capture
  effect = "none"
  because = "Under fixed contracts, consumers cannot mutate accepted syntax or graph facts. A schema extension counts as a checked-graph change."
  holds_if = [assumption.semantic_contracts]
  under = workload.semantic_changes
}

influence {
  from = [dp.binding_assessment, dp.qualified_analysis, dp.read_only_reports]
  to = fr.trace_selection
  effect = "none"
  because = "Consumers cannot rewrite candidate history or selection policy under the snapshot contract."
  holds_if = [assumption.semantic_contracts]
  under = workload.semantic_changes
}

influence {
  from = [dp.qualified_analysis, dp.read_only_reports]
  to = fr.detect_drift
  effect = "none"
  because = "Binding assessment owns comparison as well as observation. Consumers cannot redefine drift or turn unavailable observations into success."
  holds_if = [assumption.semantic_contracts]
  under = workload.semantic_changes
}

influence {
  from = [dp.read_only_reports]
  to = fr.assess_method
  effect = "none"
  because = "Presentation preserves the common assessment and cannot drop conflicts or reinterpret unknown cells."
  holds_if = [assumption.semantic_contracts]
  under = workload.semantic_changes
}

# Decomposition follows the selection of DP1.
fr "admit_syntax" {
  alias = "FR11"
  parent = dp.checked_graph
  statement = "Admit only the documented syntax and expression profile."
  criterion "admitted_expressions" {
    metric = "Malformed syntax, duplicate attributes, invalid separators, and unsupported expressions produce positioned diagnostics."
    op = "=="
    value = true
    allocated_from = fr.capture.criterion.lossless
    origin = "proposed"
  }
}

fr "resolve_identity" {
  alias = "FR12"
  parent = dp.checked_graph
  statement = "Preserve valid AD identities, typed links, and hierarchy."
  criterion "typed_graph" {
    metric = "Duplicate labels, wrong reference kinds, and cycles cannot produce a successful checked graph. Valid nested criterion references resolve."
    op = "=="
    value = true
    allocated_from = fr.capture.criterion.lossless
    origin = "proposed"
  }
}

dp "syntax_profile" {
  alias = "DP11"
  controls = [fr.admit_syntax]
  responsibility = "Parse native HCL with Go github.com/hashicorp/hcl/v2, then admit literal values and static traversals while retaining expression identity and source positions."
  decision = "selected"
  bounded_by = [constraint.native_hcl]
  owns_state = [state.syntax]
}

dp "typed_resolution" {
  alias = "DP12"
  controls = [fr.resolve_identity]
  responsibility = "Decode schema before graph construction, then resolve typed identities and hierarchy."
  decision = "selected"
  owns_state = [state.graph]
}

influence {
  from = [dp.syntax_profile]
  to = fr.admit_syntax
  effect = "material"
  mechanism = "The parser and expression admission policy decide whether each source form belongs to the admitted profile."
  mechanism_source = "inferred"
  under = workload.semantic_changes
}

influence {
  from = [dp.typed_resolution]
  to = fr.resolve_identity
  effect = "material"
  mechanism = "Schema decoding and typed resolution decide whether identities and links form a structurally valid graph."
  mechanism_source = "inferred"
  under = workload.semantic_changes
}

influence {
  from = [dp.syntax_profile]
  to = fr.resolve_identity
  effect = "material"
  mechanism = "If syntax admission loses a reference or replaces a duplicate, resolution cannot reconstruct the missing identity."
  mechanism_source = "inferred"
  under = workload.semantic_changes
}

influence {
  from = [dp.typed_resolution]
  to = fr.admit_syntax
  effect = "none"
  because = "AD decoding consumes syntax without redefining the fixed expression profile. A profile extension is a syntax-control change."
  holds_if = [assumption.fixed_expression_profile]
  under = workload.semantic_changes
}

# No compositional proof allocates DP1's aggregate effect to these leaves yet.
influence {
  from = [dp.syntax_profile]
  to = fr.trace_selection
  effect = "unknown"
  under = workload.semantic_changes
}

# Shared-resource effects do not inherit the semantic zeros.
influence {
  from = [dp.read_only_reports]
  to = fr.admit_syntax
  effect = "unknown"
  under = workload.process_failure
}

influence {
  from = [dp.binding_assessment]
  to = fr.resolve_identity
  effect = "unknown"
  under = workload.process_failure
}

state "syntax" {
  description = "Accepted syntax and positioned syntax diagnostics."
}

state "graph" {
  description = "Checked AD identities, typed links, and structural diagnostics."
}

state "snapshot" {
  description = "Candidate identity, applicable facts, context, and provenance."
}

state "drift_facts" {
  description = "Target observation provenance and comparison results. No observations are supplied."
}

state "assessment" {
  description = "Qualified conclusions, exact cells, conflicts, unknowns, and refinement obligations."
}

state "report_output" {
  description = "Rendered output and command status derived from one assessment."
}

module "syntax_admission" {
  realizes = [dp.syntax_profile]
  inputs = ["Native HCL source text subject to the documented literal-value and static-traversal expression profile."]
  outputs = ["Syntax with typed reference values and source positions, or syntax diagnostics."]
  owns_state = [state.syntax]
  dependencies = []
  invariants = ["Unsupported expressions never change meaning silently.", "Duplicate attributes never replace earlier facts."]
  failure_behavior = "Reject malformed or unsupported input with positioned diagnostics. Do not supply a successful graph."
  acceptance_checks = [fr.admit_syntax.criterion.admitted_expressions, scenario.reject_unsafe_input]
}

module "ad_resolution" {
  realizes = [dp.typed_resolution]
  inputs = ["Admitted syntax with source positions."]
  outputs = ["A checked graph or independent structural diagnostics."]
  owns_state = [state.graph]
  dependencies = [module.syntax_admission]
  invariants = ["Every accepted entity retains identity.", "A structural error cannot enter graph-dependent analysis as valid data."]
  failure_behavior = "Collect independent diagnostics and stop only analyses that require an invalid graph."
  acceptance_checks = [fr.resolve_identity.criterion.typed_graph, scenario.reject_unsafe_input]
}

module "candidate_selection" {
  realizes = [dp.candidate_snapshot]
  inputs = ["Checked graph, requested candidate, and operating context."]
  outputs = ["A candidate snapshot with provenance, or selection diagnostics."]
  owns_state = [state.snapshot]
  dependencies = [module.ad_resolution]
  invariants = ["Inactive alternatives remain available.", "All applicable records remain visible instead of last-write replacement."]
  failure_behavior = "Diagnose incoherent selection and conflicting applicable facts without silently choosing a record."
  acceptance_checks = [fr.trace_selection.criterion.traceable, scenario.change_candidate]
}

module "binding_observation" {
  realizes = [dp.binding_assessment]
  inputs = ["Candidate snapshot and external observations when an adapter exists."]
  outputs = ["Versioned drift facts or an explicit unknown result."]
  owns_state = [state.drift_facts]
  dependencies = [module.candidate_selection]
  invariants = ["Comparison belongs to this responsibility.", "Missing access never establishes conformance.", "This target contract does not assert that an adapter exists."]
  failure_behavior = "Without an adapter or observations, retain unknown with the cause and candidate identity. Do not claim verification."
  acceptance_checks = [fr.detect_drift.criterion.comparison, scenario.unavailable_observations]
}

module "ad_analysis" {
  realizes = [dp.qualified_analysis]
  inputs = ["Candidate snapshot and available drift facts. Source-only analysis does not require a successful observation."]
  outputs = ["A qualified assessment with known effects, conflicts, unknowns, and obligations."]
  owns_state = [state.assessment]
  dependencies = [module.candidate_selection, module.binding_observation]
  invariants = ["Exact effects remain visible before aggregate refinement.", "Pairings alone never prove effective diagonal controls.", "Unknown cells never become zero by omission."]
  failure_behavior = "Withdraw conclusions whose premises fail. Preserve independent findings and unresolved obligations."
  acceptance_checks = [fr.assess_method.criterion.qualified, scenario.exact_conflict, scenario.invalidate_premise]
}

module "presentation" {
  realizes = [dp.read_only_reports]
  inputs = ["One qualified assessment and command diagnostics."]
  outputs = ["Consistent text views and process status."]
  owns_state = [state.report_output]
  dependencies = [module.ad_analysis]
  invariants = ["Rendering cannot mutate source facts or assessment values.", "Every material qualification survives projection."]
  failure_behavior = "Return failure status for invalid input and failed assertions without a traceback. Do not report an unavailable assessment as success."
  acceptance_checks = [fr.present.criterion.faithful_reports, scenario.change_report, scenario.shared_failure]
}

# This flow records source analysis, not an implemented probe runner.
# Design order also constrains changes to contracts. It is not this runtime list.
flow "source_assessment" {
  owner = module.presentation
  input = "Source text, candidate request, and operating context."
  output = "Qualified source assessment and process status, without implementation verification."
  steps = [module.syntax_admission, module.ad_resolution, module.candidate_selection, module.ad_analysis, module.presentation]
}

decision "assessment_contracts" {
  addresses = [
    fr.qualified_assessment,
    fr.capture,
    fr.trace_selection,
    fr.detect_drift,
    fr.assess_method,
    fr.present,
    fr.admit_syntax,
    fr.resolve_identity,
  ]
  workload = workload.semantic_changes
  context = context.semantic_changes
  horizon = "One assessment of a versioned design corpus. No probability observations exist."
  candidate "go_hcl_target" {
    dps = [dp.assessment_pipeline, dp.checked_graph, dp.candidate_snapshot, dp.binding_assessment, dp.qualified_analysis, dp.read_only_reports, dp.syntax_profile, dp.typed_resolution]
    workload = workload.semantic_changes
    version = "proposed-go-contracts-2026-10-29"
    information {
      basis = "unknown"
    }
  }
  selected = "go_hcl_target"
  provisional = true
  why = "Select the planned Go HCL tool under explicit assessment contracts. Comparable joint evidence is absent, so this is not an Information Axiom ranking."
}

scenario "reject_unsafe_input" {
  kind = "failure"
  verifies = [fr.admit_syntax.criterion.admitted_expressions, fr.resolve_identity.criterion.typed_graph]
  vary = [dp.syntax_profile, dp.typed_resolution]
  description = "Proposed acceptance scenario, not a completed run."
  steps = ["Introduce a duplicate attribute, duplicate label, wrong reference kind, or hierarchy cycle.", "Require positioned diagnostics before dependent analysis."]
}

scenario "change_candidate" {
  kind = "change"
  verifies = [fr.trace_selection.criterion.traceable]
  vary = [dp.candidate_snapshot]
  expect_affected = [fr.trace_selection, fr.detect_drift, fr.assess_method, fr.present]
  description = "Proposed scenario. Preserve alternatives and invalidate previous candidate results."
  steps = ["Select a different candidate in a versioned fixture.", "Require new binding applicability, assessment, and reports without deleting history."]
}

scenario "unavailable_observations" {
  kind = "failure"
  verifies = [fr.detect_drift.criterion.comparison]
  vary = [dp.binding_assessment]
  description = "No adapter exists here. A future seeded-mismatch trial is separate from this missing-observation case."
  steps = ["Request binding assessment without an adapter or observations.", "Require unknown rather than verified implementation."]
}

scenario "exact_conflict" {
  kind = "change"
  verifies = [fr.assess_method.criterion.qualified]
  vary = [dp.qualified_analysis]
  description = "Proposed fixture change, not an authored contradiction in this example."
  steps = ["Add an exact leaf effect beside an ancestor zero.", "Require the exact effect and an unresolved refinement obligation in master, impact, and DSM views.", "Report contradiction only when an applicable outcome argument establishes incompatible claims."]
}

scenario "invalidate_premise" {
  kind = "change"
  verifies = [fr.assess_method.criterion.qualified]
  vary = [dp.qualified_analysis]
  steps = ["Invalidate the semantic-contract premise in a fixture.", "Withdraw supported independence and expose the affected obligations."]
}

scenario "change_report" {
  kind = "change"
  verifies = [fr.present.criterion.faithful_reports]
  vary = [dp.read_only_reports]
  expect_unaffected = [fr.capture, fr.trace_selection, fr.detect_drift, fr.assess_method]
  under = workload.semantic_changes
  description = "This proposed zero-effect test depends on the unvalidated semantic-contract premise."
  steps = ["Change formatting without changing the assessment.", "Compare all values, conditions, findings, and failure status across views."]
}

scenario "shared_failure" {
  kind = "failure"
  verifies = [fr.present.criterion.faithful_reports]
  vary = [dp.read_only_reports]
  under = workload.process_failure
  description = "Availability bounds remain unspecified. No isolation or recovery result is claimed."
  steps = ["Consider shared process memory or file access failure.", "Require an unavailable assessment rather than a successful or verified result."]
}

# Language-level extension design. This is a proposed local diagonal.
# It does not remove the core pipeline's upstream influences.
assumption "fixed_owner_contracts" {
  statement = "Core identities and role contracts stay fixed. Owners cannot change core facts, shared policy, or another owner's inputs and results. Resource-failure effects remain outside this semantic context."
  status = "to_validate"
}

fr "extension_results" {
  need = [need.tool_exchange, need.faithful_design]
  statement = "Retain separate, qualified results from external model owners."
  criterion "separate_results" {
    metric = "An owner failure preserves independent core and owner findings."
    op = "=="
    value = true
    origin = "proposed"
  }
}

dp "owner_envelopes" {
  controls = [fr.extension_results]
  responsibility = "Compose immutable core results and separately identified owner outcomes without rewriting either."
  decision = "selected"
}

fr "binding_links" {
  parent = dp.owner_envelopes
  statement = "Distinguish a matching realization from a mismatch or unavailable observation."
  criterion "binding_status" {
    metric = "Caller-supplied observations produce match, mismatch, or unknown under the binding contract."
    op = "=="
    value = true
    origin = "proposed"
  }
}

fr "behavior_links" {
  parent = dp.owner_envelopes
  statement = "Identify an external behavior target at its exact revision."
  criterion "target_status" {
    metric = "The owner reports resolved, missing, or unknown without substituting revisions."
    op = "=="
    value = true
    origin = "proposed"
  }
}

fr "view_layout" {
  parent = dp.owner_envelopes
  statement = "Select report layout while preserving every finding and qualification."
  criterion "preserved_results" {
    metric = "Every accepted layout presents identical assessment values and operation status."
    op = "=="
    value = true
    origin = "proposed"
  }
}

dp "binding_owner" {
  controls = [fr.binding_links]
  responsibility = "Decode binding data and compare its address against the named caller-supplied observation set."
  decision = "selected"
}

dp "behavior_owner" {
  controls = [fr.behavior_links]
  responsibility = "Resolve external model, revision, target kind, and label under a separately versioned contract."
  decision = "selected"
}

dp "view_owner" {
  controls = [fr.view_layout]
  responsibility = "Decode presentation choices without changing assessment facts or suppressing findings."
  decision = "selected"
}

influence {
  from = [dp.owner_envelopes]
  to = fr.extension_results
  effect = "material"
  mechanism = "Separate result envelopes retain independent findings when an owner fails."
  mechanism_source = "inferred"
}

influence {
  from = [dp.binding_owner]
  to = fr.binding_links
  effect = "material"
  mechanism = "The binding contract determines comparison outcomes."
  mechanism_source = "inferred"
}

influence {
  from = [dp.behavior_owner]
  to = fr.behavior_links
  effect = "material"
  mechanism = "The behavior contract determines exact target identity."
  mechanism_source = "inferred"
}

influence {
  from = [dp.view_owner]
  to = fr.view_layout
  effect = "material"
  mechanism = "The view contract determines presentation choices and preserves the supplied findings."
  mechanism_source = "inferred"
}

influence {
  from = [dp.behavior_owner, dp.view_owner]
  to = fr.binding_links
  effect = "none"
  because = "Separate owner inputs and immutable results prevent either owner from changing binding comparison."
  holds_if = [assumption.fixed_owner_contracts]
  under = workload.semantic_changes
}

influence {
  from = [dp.binding_owner, dp.view_owner]
  to = fr.behavior_links
  effect = "none"
  because = "Neither owner can change behavior resolution inputs or policy under fixed contracts."
  holds_if = [assumption.fixed_owner_contracts]
  under = workload.semantic_changes
}

influence {
  from = [dp.binding_owner, dp.behavior_owner]
  to = fr.view_layout
  effect = "none"
  because = "Changing owner data cannot change layout policy. Assessment values remain supplied inputs under the fixed contract."
  holds_if = [assumption.fixed_owner_contracts]
  under = workload.semantic_changes
}

scenario "isolate_owner_change" {
  kind = "change"
  verifies = [fr.extension_results.criterion.separate_results]
  vary = [dp.binding_owner]
  expect_affected = [fr.binding_links]
  expect_unaffected = [fr.behavior_links, fr.view_layout]
  under = workload.semantic_changes
  description = "Proposed acceptance scenario. No completed execution or isolation evidence is claimed."
  steps = ["Change only binding data under fixed contracts.", "Require identical behavior resolution and layout policy.", "Fail the binding owner and retain independent findings."]
}
