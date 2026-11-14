resource "opsolete_variable_set_project" "azure_prd" {
  variable_set_id = opsolete_variable_set.azure_prd.id
  project_id      = opsolete_project.prd.id
}
