attachment "admission_configuration" {
  owner = extension.terraform_binding
  subject = pv.admission_policy
  data = {
    address = "terraform_data.admission_policy"
    observation_source = "receipt_configuration"
  }
}
