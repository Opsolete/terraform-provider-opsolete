# An empty workspace; its templates are added in Opsolete's UI.
resource "opsolete_workspace" "shop" {
  name            = "shop-prd"
  project_id      = opsolete_project.prd.id
  drift_detection = true
}
