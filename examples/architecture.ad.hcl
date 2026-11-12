# Architecture example: a pallet batch counter with one triangular mapping.
# Run: ./bin/adhcl validate --profile architecture examples/architecture.ad.hcl
design "pallet_count" {
  boundary = "Count the accepted scans of one pallet batch. The scanner hardware, the network, and the storage system are outside the boundary."
  level = "architecture"
}

need "valid_scans_only" {
  statement = "Count only scans that have exactly twelve digits."
  why = "A scan with another format is not a valid product code."
  origin = "proposed"
}

need "exact_count" {
  statement = "Report the exact number of counted scans for each batch."
  origin = "proposed"
}

state "accepted_scan" {
  description = "The scan text that passed the digit rule for one batch. It does not change after it is published."
}

state "batch_count" {
  description = "The number of accepted scans in the current batch."
}

fr "accept_scan" {
  need = [need.valid_scans_only]
  statement = "Accept a scan only when it has exactly twelve digits."
  criterion "digit_count" {
    statement = "Every accepted scan has exactly twelve digits."
    metric = "number of digits in an accepted scan"
    op = "=="
    value = 12
    unit = "digits"
    origin = "proposed"
  }
}

fr "count_pallet" {
  need = [need.exact_count]
  statement = "Return the number of accepted scans in the batch."
  criterion "exact_count" {
    statement = "The returned count equals the number of accepted scans."
    metric = "absolute difference between the returned count and the number of accepted scans"
    op = "=="
    value = 0
    unit = "scans"
    origin = "proposed"
  }
}

dp "scan_admission" {
  controls = [fr.accept_scan]
  responsibility = "Accept a scan only when its text has exactly twelve digits, then publish it as the accepted scan."
  decision = "selected"
  owns_state = [state.accepted_scan]
}

dp "batch_counter" {
  controls = [fr.count_pallet]
  responsibility = "Add one to the batch count for each accepted scan."
  decision = "selected"
  owns_state = [state.batch_count]
  reads_state = [state.accepted_scan]
}

influence {
  from = [dp.scan_admission]
  to = fr.accept_scan
  effect = "material"
  mechanism = "The digit rule decides which scans are accepted."
}

influence {
  from = [dp.batch_counter]
  to = fr.count_pallet
  effect = "material"
  mechanism = "The counting rule decides the value of the returned count."
}

influence {
  from = [dp.scan_admission]
  to = fr.count_pallet
  effect = "material"
  mechanism = "The count includes only accepted scans, so a change to the digit rule changes the count."
}

influence {
  from = [dp.batch_counter]
  to = fr.accept_scan
  effect = "none"
  because = "Adding a scan to the count does not change how many digits the scan has."
}

expect "pallet_structure" {
  scope = design.pallet_count
  classification = "decoupled"
  order = [dp.scan_admission, dp.batch_counter]
  basis = "source"
}

module "scan_check" {
  realizes = [dp.scan_admission]
  inputs = ["One scan text value from the scanner."]
  outputs = ["An accepted scan, or a rejection without any scan text."]
  owns_state = [state.accepted_scan]
  dependencies = []
  invariants = ["Only text with exactly twelve digits becomes an accepted scan.", "A published accepted scan does not change."]
  failure_behavior = "Reject the scan and publish nothing when the scan text cannot be read."
  acceptance_checks = [fr.accept_scan.criterion.digit_count, scenario.scan_digits]
}

module "batch_count" {
  realizes = [dp.batch_counter]
  inputs = ["The accepted scans of one batch, read from the scan module."]
  outputs = ["The number of accepted scans in the batch."]
  owns_state = [state.batch_count]
  dependencies = [module.scan_check]
  invariants = ["The count equals the number of accepted scans added to the batch.", "A rejected scan never changes the count."]
  failure_behavior = "Report a failure and no count when an accepted scan cannot be read."
  acceptance_checks = [fr.count_pallet.criterion.exact_count, scenario.batch_counts]
}

scenario "scan_digits" {
  kind = "operational"
  verifies = [fr.accept_scan.criterion.digit_count]
  description = "Covers the digit rule at the accepted length and at the lengths around it."
  steps = [
    "Submit a scan with twelve digits and expect it to be accepted.",
    "Submit scans with eleven digits, thirteen digits, and letters. Expect a rejection for each.",
    "Expect no accepted scan from any rejected input.",
  ]
}

scenario "batch_counts" {
  kind = "operational"
  verifies = [fr.count_pallet.criterion.exact_count]
  description = "Covers a batch with a rejected scan and an empty batch."
  steps = [
    "Submit three accepted scans and one rejected scan to one batch.",
    "Expect the returned count to be 3.",
    "Submit an empty batch and expect the returned count to be 0.",
  ]
}
