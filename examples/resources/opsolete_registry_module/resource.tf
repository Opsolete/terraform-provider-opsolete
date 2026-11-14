# Every tag storage-account-v* of the repository becomes a version of
# <host>/terraform-modules-examples/storage-account/azurerm.
resource "opsolete_registry_module" "storage_account" {
  vcs_connection_id = opsolete_vcs_connection.github.id
  repo              = "Opsolete/terraform-modules-examples"
  dir               = "storage-account"
  tag_prefix        = "storage-account-"
  namespace         = "terraform-modules-examples"
  name              = "storage-account"
  system            = "azurerm"
}
