# The templates live in an Azure DevOps repository; the builder commits to it.
resource "opsolete_template_catalog" "this" {
  vcs_connection_id = opsolete_vcs_connection.ado.id
  repo              = "Infra/templates"
  branch            = "main"
  mode              = "commit"
}
