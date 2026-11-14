resource "opsolete_workspace_variable" "tags" {
  workspace_id = opsolete_workspace.shop.id
  category     = "terraform"
  key          = "tags"
  value        = "{team = \"shop\"}"
  hcl          = true
}
