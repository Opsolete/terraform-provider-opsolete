# Every workspace of the project deploys this template in its subscription's region.
resource "opsolete_environment_value" "region" {
  project_id  = opsolete_project.prd.id
  template_id = "storage-account"
  input       = "location"
  value       = "swedencentral"
}
