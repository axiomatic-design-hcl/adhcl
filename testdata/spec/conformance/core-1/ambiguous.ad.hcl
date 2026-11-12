# Two complete control pairings exist. Matrix appearance cannot choose one.
design "ambiguous" {
  boundary = "A synthetic control-pairing acceptance case, not a recommended architecture."
  level = "architecture"
}

need "two_outcomes" {
  statement = "Receive both configured output strings exactly."
  origin = "inferred"
}

fr "first" {
  need = [need.two_outcomes]
  statement = "Return the exact first configured string."
  criterion "exact" {
    metric = "The returned first string equals the configured first string byte for byte."
    op = "=="
    value = true
    origin = "proposed"
  }
}

fr "second" {
  need = [need.two_outcomes]
  statement = "Return the exact second configured string."
  criterion "exact" {
    metric = "The returned second string equals the configured second string byte for byte."
    op = "=="
    value = true
    origin = "proposed"
  }
}

dp "first_control" {
  controls = [fr.first, fr.second]
  responsibility = "Select a shared decoding rule for both configured strings."
  decision = "selected"
}

dp "second_control" {
  controls = [fr.first, fr.second]
  responsibility = "Select a shared encoding rule for both output strings."
  decision = "selected"
}

influence {
  from = [dp.first_control, dp.second_control]
  to = fr.first
  effect = "material"
  mechanism = "Changing either shared rule changes the first output bytes."
  mechanism_source = "inferred"
}

influence {
  from = [dp.first_control, dp.second_control]
  to = fr.second
  effect = "material"
  mechanism = "Changing either shared rule changes the second output bytes."
  mechanism_source = "inferred"
}

module "both_outputs" {
  realizes = [dp.first_control, dp.second_control]
  inputs = ["Two configured strings and their shared encoding and decoding rules."]
  outputs = ["Two output strings."]
  owns_state = []
  dependencies = []
  invariants = ["Each output must preserve the bytes of its corresponding configured string."]
  failure_behavior = "Do not claim exact equality after a failed conversion."
  acceptance_checks = [fr.first.criterion.exact, fr.second.criterion.exact, scenario.two_strings]
}

scenario "two_strings" {
  kind = "operational"
  verifies = [fr.first.criterion.exact, fr.second.criterion.exact]
  description = "A proposed check, not an observed result."
  steps = ["Supply distinct strings including non-ASCII bytes.", "Require each returned string to equal its corresponding configured string exactly."]
}
