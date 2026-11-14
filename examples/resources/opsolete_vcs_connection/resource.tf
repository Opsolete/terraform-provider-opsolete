resource "opsolete_vcs_connection" "ado" {
  name          = "azure-devops"
  provider_type = "azure-devops"
  base_url      = "https://dev.azure.com/my-org"
  token         = var.ado_token
  scope         = "selected"
  project_ids   = [opsolete_project.prd.id]
}
