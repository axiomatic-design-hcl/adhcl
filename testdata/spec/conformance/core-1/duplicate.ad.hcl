# This label already exists in base.ad.hcl. File order must not replace it.
need "separate_outputs" {
  statement = "A second declaration must not win by file order."
}
