attachment "list_module_model" {
  owner = extension.event_model_link
  subject = module.list_check
  data = {
    model = "badge_gate_behavior"
    revision = "badge-gate-proposal-1"
    target_kind = "slice"
    target_label = "decide_badge"
  }
}
