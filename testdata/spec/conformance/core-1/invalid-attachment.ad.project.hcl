project "invalid_attachment" {
  language = "requirements_hcl/core/1"
  core_files = ["base.ad.hcl"]
  attachment_files = ["invalid-view.ad.ext.hcl"]
}

extension "html_view" {
  role = "view"
  contract = "requirements_hcl/html_view/1"
}
