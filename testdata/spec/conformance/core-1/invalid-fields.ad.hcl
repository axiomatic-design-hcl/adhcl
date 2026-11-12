# Every construct parses as native HCL. The field contracts must reject it.
source "null_title" {
  title = null
}

need "unknown_field" {
  statement = "Retain an unknown field as an error."
  ignored_typo = true
}


dp "empty_controls" {
  controls = []
  responsibility = "An empty list cannot name a proposed control."
  decision = "proposed"
}

scenario "string_reference" {
  kind = "operational"
  verifies = ["fr.left.criterion.exact"]
  steps = ["A string that resembles a reference must not become a traversal."]
}
