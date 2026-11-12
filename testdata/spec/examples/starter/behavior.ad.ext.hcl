attachment "total_behavior" {
  owner = extension.event_model_link
  subject = module.total
  data = {
    model = "receipt_behavior"
    revision = "receipt-behavior-proposal-1"
    target_kind = "slice"
    target_label = "calculate_total"
  }
}
