project "wrong_kind" {
  language = "requirements_hcl/core/1"
  core_files = ["base.ad.hcl", "wrong-kind.ad.hcl"]
  attachment_files = []
}
