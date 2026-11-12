# Acceptance fixture only. These proposed controls have no reviewed support.
design "two_outputs" {
  boundary = "Return two independent configured strings under a fixed immutable input contract. Process failures are outside the change range."
  level = "architecture"
}

need "separate_outputs" {
  statement = "Receive the exact configured left and right strings."
  origin = "inferred"
}

fr "left" {
  need = [need.separate_outputs]
  statement = "Return the configured left string unchanged."
  criterion "exact" {
    metric = "For every supplied left string, the returned left string is byte-for-byte equal to it."
    op = "=="
    value = true
    origin = "proposed"
  }
}

fr "right" {
  need = [need.separate_outputs]
  statement = "Return the configured right string unchanged."
  criterion "exact" {
    metric = "For every supplied right string, the returned right string is byte-for-byte equal to it."
    op = "=="
    value = true
    origin = "proposed"
  }
}

dp "left_output" {
  controls = [fr.left]
  responsibility = "Copy the immutable left input to the left output without reading or changing the right input."
  decision = "selected"
}

dp "right_output" {
  controls = [fr.right]
  responsibility = "Copy the immutable right input to the right output without reading or changing the left input."
  decision = "selected"
}

influence {
  from = [dp.left_output]
  to = fr.left
  effect = "material"
  mechanism = "Changing the left-copy rule changes whether the exact-left predicate holds."
  mechanism_source = "inferred"
}

influence {
  from = [dp.right_output]
  to = fr.right
  effect = "material"
  mechanism = "Changing the right-copy rule changes whether the exact-right predicate holds."
  mechanism_source = "inferred"
}

cells "separate_inputs" {
  scope = design.two_outputs
  unlisted = "none"
  because = "Within the fixed immutable-input contract, each copy rule reads only its own input and cannot change the other rule or input. This is a proposed semantic boundary, not measured isolation."
}

module "left" {
  realizes = [dp.left_output]
  inputs = ["An immutable left string."]
  outputs = ["The same left string."]
  owns_state = []
  dependencies = []
  invariants = ["The right input and output are not read or written."]
  failure_behavior = "Do not publish a different string as a successful copy. Process availability is outside this contract."
  acceptance_checks = [fr.left.criterion.exact, scenario.copy_strings]
}

module "right" {
  realizes = [dp.right_output]
  inputs = ["An immutable right string."]
  outputs = ["The same right string."]
  owns_state = []
  dependencies = []
  invariants = ["The left input and output are not read or written."]
  failure_behavior = "Do not publish a different string as a successful copy. Process availability is outside this contract."
  acceptance_checks = [fr.right.criterion.exact, scenario.copy_strings]
}

scenario "copy_strings" {
  kind = "operational"
  verifies = [fr.left.criterion.exact, fr.right.criterion.exact]
  description = "Proposed checks only; no observations are supplied."
  steps = ["Supply empty strings, different strings, and strings containing non-ASCII characters.", "Require each output to equal its own input byte for byte.", "Vary each input independently and require the other output to remain unchanged."]
}
