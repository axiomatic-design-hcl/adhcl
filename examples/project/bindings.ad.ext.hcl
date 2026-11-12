attachment "list_lookup_config" {
  owner = extension.terraform_binding
  subject = pv.active_list_lookup
  data = {
    address = "terraform_data.badge_list_lookup"
    observation_source = "badge_gate_configuration"
  }
}
