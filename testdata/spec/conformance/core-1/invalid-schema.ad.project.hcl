project "invalid_schema" {
  language = "requirements_hcl/core/1"
  core_files = ["base.ad.hcl", "invalid-fields.ad.hcl"]
  attachment_files = []
}
