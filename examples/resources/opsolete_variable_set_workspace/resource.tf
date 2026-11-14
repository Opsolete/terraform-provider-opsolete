# A repository-linked workspace gets only sets attached to it.
resource "opsolete_variable_set_workspace" "azure_prd" {
  variable_set_id = opsolete_variable_set.azure_prd.id
  workspace_id    = opsolete_workspace.shop.id
}
