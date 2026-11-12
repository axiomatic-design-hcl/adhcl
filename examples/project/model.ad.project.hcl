# Project example manifest: one core file and one attachment for each standard extension.
# Run: ./bin/adhcl validate examples/project/model.ad.project.hcl
project "badge_gate" {
  language = "requirements_hcl/core/1"
  core_files = ["design.ad.hcl"]
  attachment_files = ["bindings.ad.ext.hcl", "behavior.ad.ext.hcl", "views.ad.ext.hcl"]
}

extension "terraform_binding" {
  role = "binding"
  contract = "requirements_hcl/terraform_binding/1"
}

extension "event_model_link" {
  role = "behavior_link"
  contract = "requirements_hcl/event_model_link/1"
}

extension "html_view" {
  role = "view"
  contract = "requirements_hcl/html_view/1"
}
