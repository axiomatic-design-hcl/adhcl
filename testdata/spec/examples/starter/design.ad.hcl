# Proposed architecture, not implemented code or measured evidence.
# This semantic-change range excludes process failure and resource exhaustion.
design "receipt" {
  boundary = "Admit one amount in integer cents and calculate its total with a configured fee. Payment, storage, deployment, and process availability are outside the boundary."
  level = "architecture"
  workload = workload.receipt_inputs
}

workload "receipt_inputs" {
  description = "One amount and one fee per request. Inputs outside the admitted amount range must be rejected. A fee is fixed for a request. No latency or availability target is claimed."
  ranges = {
    amount_cents = { min = 0, max = 100000, unit = "cents" }
    fee_cents = { min = 0, max = 1000, unit = "cents" }
  }
}

assumption "fixed_contracts" {
  statement = "During a semantic change, the integer-cent input contract and immutable admission result remain fixed. The total module cannot change admission policy or rewrite its result. Shared process failures are outside this change range."
  status = "to_validate"
}

need "correct_receipt" {
  statement = "Receive a correct total without treating invalid amounts as accepted amounts."
  why = "A caller must distinguish invalid input from a valid receipt."
  origin = "inferred"
}

fr "admit_amount" {
  need = [need.correct_receipt]
  statement = "Accept exactly the permitted integer-cent amounts."
  criterion "admission" {
    metric = "An amount is admitted if and only if it is an integer from 0 through 100000 inclusive; every other input returns invalid_amount and no admitted amount."
    op = "=="
    value = true
    origin = "proposed"
  }
}

fr "calculate_total" {
  need = [need.correct_receipt]
  statement = "Return the exact total for an admitted amount, or preserve the invalid-input result."
  criterion "total" {
    metric = "For every admitted amount and integer fee from 0 through 1000 inclusive, total_cents equals amount_cents plus fee_cents exactly. An invalid admission returns invalid_amount and no total."
    op = "=="
    value = true
    origin = "proposed"
  }
}

state "admitted_amount" {
  description = "The request-local immutable admitted integer amount or invalid_amount result."
}

state "total_result" {
  description = "The request-local total or invalid_amount result."
}

dp "amount_admission" {
  controls = [fr.admit_amount]
  responsibility = "Check integer type and the inclusive amount range before publishing an immutable admission result."
  decision = "selected"
  owns_state = [state.admitted_amount]
  holds_if = [assumption.fixed_contracts]
}

dp "exact_addition" {
  controls = [fr.calculate_total]
  responsibility = "Read the immutable admission result, propagate invalid_amount, and otherwise add the request's integer fee without rounding or overflow in the stated range."
  decision = "selected"
  owns_state = [state.total_result]
  reads_state = [state.admitted_amount]
  holds_if = [assumption.fixed_contracts]
}

influence {
  from = [dp.amount_admission]
  to = fr.admit_amount
  effect = "material"
  mechanism = "Admission policy determines which inputs are accepted or rejected."
  mechanism_source = "inferred"
}

influence {
  from = [dp.exact_addition]
  to = fr.calculate_total
  effect = "material"
  mechanism = "The addition and invalid-result propagation rules determine whether the returned total satisfies its exact predicate."
  mechanism_source = "inferred"
}

influence {
  from = [dp.amount_admission]
  to = fr.calculate_total
  effect = "material"
  mechanism = "Accepting a noninteger or rejecting a permitted amount changes whether a correct receipt can be returned. A changed admission contract requires renewed total acceptance checks."
  mechanism_source = "inferred"
}

cells "fixed_admission" {
  scope = design.receipt
  unlisted = "none"
  because = "Within the fixed immutable-result contract, changing the downstream addition policy cannot alter amount admission. This default states a proposed semantic boundary, not measured isolation."
}

pv "admission_policy" {
  realizes = [dp.amount_admission]
  kind = "configuration"
  note = "A proposed realization subject for the binding attachment. No implementation or Terraform observation is supplied."
}

pv "total_routine" {
  realizes = [dp.exact_addition]
  kind = "routine"
  note = "A proposed exact integer addition routine. No code or execution evidence is supplied."
}

module "admission" {
  realizes = [dp.amount_admission]
  inputs = ["One untrusted amount value."]
  outputs = ["An immutable admitted integer amount or invalid_amount without an amount."]
  owns_state = [state.admitted_amount]
  dependencies = []
  invariants = ["No invalid value becomes an admitted amount.", "A published admission result cannot be rewritten by consumers."]
  failure_behavior = "Reject values outside the integer-cent contract with invalid_amount. Do not supply an amount after rejection. Process availability is outside this contract."
  acceptance_checks = [fr.admit_amount.criterion.admission, scenario.amount_boundaries]
}

module "total" {
  realizes = [dp.exact_addition]
  inputs = ["The immutable admission result and an integer fee from 0 through 1000 cents."]
  outputs = ["The exact integer total or invalid_amount without a total."]
  owns_state = [state.total_result]
  dependencies = [module.admission]
  invariants = ["Admission results remain unchanged.", "The stated integer range requires no rounding and fits the selected arithmetic representation."]
  failure_behavior = "Propagate invalid_amount without calculating a total. Never replace invalid input with zero. An out-of-contract fee must be rejected rather than used to claim the total predicate."
  acceptance_checks = [fr.calculate_total.criterion.total, scenario.total_boundaries]
}

flow "receipt_request" {
  owner = module.total
  input = "An untrusted amount and an in-range integer fee."
  output = "An exact total or invalid_amount."
  steps = [module.admission, module.total]
}

scenario "amount_boundaries" {
  kind = "operational"
  verifies = [fr.admit_amount.criterion.admission]
  description = "Proposed checks for the admission leaf. These are instructions, not recorded observations."
  steps = ["Admit 0 and 100000 integer cents.", "Reject -1, 100001, 1.5, a string, and null without publishing an amount.", "For each remaining integer in the permitted range, require admission."]
}

scenario "total_boundaries" {
  kind = "operational"
  verifies = [fr.calculate_total.criterion.total]
  description = "Proposed checks for the total leaf. No completed trial is claimed."
  steps = ["For each admitted amount and in-range integer fee, require their exact integer sum.", "Include 0 plus 0 and 100000 plus 1000.", "Pass an invalid admission result and require invalid_amount with no total.", "Require the admission result to remain unchanged after calculation."]
}

scenario "change_addition_policy" {
  kind = "change"
  verifies = [fr.admit_amount.criterion.admission, fr.calculate_total.criterion.total]
  vary = [dp.exact_addition]
  expect_affected = [fr.calculate_total]
  expect_unaffected = [fr.admit_amount]
  description = "Proposed change check under the fixed immutable admission contract. No change or isolation result has been measured."
  steps = ["Compare two addition-policy variants while holding the admission policy, input range, and immutable admission results fixed.", "Require identical admission decisions and unchanged published admission results.", "Re-evaluate the exact-total predicate for each addition variant rather than assuming the changed policy still satisfies it."]
}

scenario "propagate_invalid_admission" {
  kind = "failure"
  verifies = [fr.admit_amount.criterion.admission, fr.calculate_total.criterion.total]
  description = "Proposed failure check for a rejected amount. No execution result is supplied."
  steps = ["Reject -1, 1.5, and a string as invalid_amount with no admitted amount.", "Pass each immutable invalid_amount result to the total module with an in-range fee.", "Require invalid_amount with no total; do not fabricate a zero amount, a zero total, or a fee-only total.", "Require the original admission result to remain unchanged."]
}

decision "receipt_mechanism" {
  addresses = [fr.admit_amount, fr.calculate_total]
  workload = workload.receipt_inputs
  horizon = "One request under fixed input and result contracts."
  candidate "integer_cents" {
    dps = [dp.amount_admission, dp.exact_addition]
    version = "receipt-proposal-1"
    information {
      basis = "unknown"
    }
  }
  selected = "integer_cents"
  provisional = true
  why = "Select explicit admission followed by exact integer addition provisionally. No reviewed support or joint empirical success evidence is supplied."
}
