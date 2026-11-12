# Project example core: a badge gate with one triangular mapping.
design "badge_gate" {
  boundary = "Decide admit or deny for one badge scan and write one log entry for each decision. The card reader, the door lock, and the log archive are outside the boundary."
  level = "architecture"
}

need "only_listed_badges" {
  statement = "Admit only badges that are on the active list."
  why = "A door must open only for people who may enter."
  origin = "proposed"
}

need "complete_log" {
  statement = "Record every admit and deny decision."
  origin = "proposed"
}

state "lookup_decision" {
  description = "The admit or deny decision for one badge scan."
}

state "log_record" {
  description = "The log entry written for one decision."
}

fr "match_badge" {
  need = [need.only_listed_badges]
  statement = "Admit a badge only when its identifier is on the active list."
  criterion "listed_only" {
    statement = "No admitted badge identifier is missing from the active list."
    metric = "number of admitted badge identifiers that are not on the active list"
    op = "=="
    value = 0
    unit = "badges"
    origin = "proposed"
  }
}

fr "log_entry" {
  need = [need.complete_log]
  statement = "Write one log entry for each admit or deny decision."
  criterion "one_entry" {
    statement = "Every decision has exactly one log entry."
    metric = "number of decisions without exactly one log entry"
    op = "=="
    value = 0
    unit = "decisions"
    origin = "proposed"
  }
}

dp "list_lookup" {
  controls = [fr.match_badge]
  responsibility = "Compare the badge identifier with the active list and publish one admit or deny decision."
  decision = "selected"
  owns_state = [state.lookup_decision]
}

dp "log_writer" {
  controls = [fr.log_entry]
  responsibility = "Append one log entry for each decision that it receives."
  decision = "selected"
  owns_state = [state.log_record]
  reads_state = [state.lookup_decision]
}

influence {
  from = [dp.list_lookup]
  to = fr.match_badge
  effect = "material"
  mechanism = "The lookup result decides whether the badge is admitted."
}

influence {
  from = [dp.log_writer]
  to = fr.log_entry
  effect = "material"
  mechanism = "The writer creates each entry, so its append rule decides the entry count."
}

influence {
  from = [dp.list_lookup]
  to = fr.log_entry
  effect = "material"
  mechanism = "Each log entry records the decision that the lookup produced, so a change to the decision changes the entry content."
}

influence {
  from = [dp.log_writer]
  to = fr.match_badge
  effect = "none"
  because = "Appending a log entry does not change whether the badge is on the list."
}

expect "gate_structure" {
  scope = design.badge_gate
  classification = "decoupled"
  order = [dp.list_lookup, dp.log_writer]
  basis = "source"
}

pv "active_list_lookup" {
  realizes = [dp.list_lookup]
  kind = "configuration"
  note = "A proposed lookup configuration for the active list. No reader observation is supplied."
}

module "list_check" {
  realizes = [dp.list_lookup]
  inputs = ["One badge identifier from the reader."]
  outputs = ["One admit or deny decision."]
  owns_state = [state.lookup_decision]
  dependencies = []
  invariants = ["A badge identifier that is not on the active list is never admitted.", "A published decision does not change."]
  failure_behavior = "Deny the badge when the active list cannot be read."
  acceptance_checks = [fr.match_badge.criterion.listed_only, scenario.list_matches]
}

module "entry_writer" {
  realizes = [dp.log_writer]
  inputs = ["One decision from the list module."]
  outputs = ["One appended log entry."]
  owns_state = [state.log_record]
  dependencies = [module.list_check]
  invariants = ["Every decision produces exactly one log entry.", "Writing a log entry does not change the decision."]
  failure_behavior = "Report a failed write and leave the decision unchanged."
  acceptance_checks = [fr.log_entry.criterion.one_entry, scenario.entry_counts]
}

scenario "list_matches" {
  kind = "operational"
  verifies = [fr.match_badge.criterion.listed_only]
  description = "Covers a listed badge, an unlisted badge, and a badge removed from the list."
  steps = [
    "Submit a badge identifier that is on the active list and expect an admit decision.",
    "Submit a badge identifier that is not on the active list and expect a deny decision.",
    "Submit an identifier removed from the list and expect a deny decision.",
  ]
}

scenario "entry_counts" {
  kind = "operational"
  verifies = [fr.log_entry.criterion.one_entry]
  description = "Covers one admitted scan, one denied scan, and a failed log write."
  steps = [
    "Submit one admitted badge and one denied badge.",
    "Expect exactly two log entries, one for each decision.",
    "Force the log write to fail and expect a reported failure with the decision unchanged.",
  ]
}
