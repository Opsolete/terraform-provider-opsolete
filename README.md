# terraform-provider-opsolete

Manage [Opsolete](https://github.com/Opsolete/Opsolete)'s platform objects with Terraform: projects and their
environment values, workspaces and their variables, variable sets (and where they apply) and VCS connections.
Templates on workspaces, runs and team access stay in Opsolete's UI.

```hcl
terraform {
  required_providers {
    opsolete = { source = "opsolete/opsolete" }
  }
}

provider "opsolete" {
  url   = "https://opsolete.example.com"
  token = var.opsolete_token # a service token; or OPSOLETE_URL and OPSOLETE_TOKEN
}

resource "opsolete_project" "prd" {
  name        = "prd"
  runner_pool = "prd"
}

resource "opsolete_variable_set" "azure_prd" { name = "azure-prd" }

resource "opsolete_variable_set_variable" "secret" {
  variable_set_id = opsolete_variable_set.azure_prd.id
  category        = "env"
  key             = "ARM_CLIENT_SECRET"
  value           = var.arm_client_secret
  sensitive       = true
}

resource "opsolete_variable_set_project" "azure_prd" {
  variable_set_id = opsolete_variable_set.azure_prd.id
  project_id      = opsolete_project.prd.id
}

resource "opsolete_workspace" "shop" {
  name       = "shop-prd"
  project_id = opsolete_project.prd.id
}
```

**The token** is a service token: Settings → API tokens in Opsolete, or `opsolete api-token terraform` on the server
(it has admin rights and is audited as `token:<name>`). Needs Opsolete with the `/api/v1` platform objects (after
0.9.0).

Resources: `opsolete_project`, `opsolete_environment_value`, `opsolete_workspace`, `opsolete_workspace_variable`,
`opsolete_variable_set`, `opsolete_variable_set_variable`, `opsolete_variable_set_project`,
`opsolete_variable_set_workspace`, `opsolete_vcs_connection`. Data sources: `opsolete_project`,
`opsolete_workspace`, `opsolete_variable_set`. All importable (by name where they have one). Reference: [docs](docs).

Good to know:

- **States are Opsolete's alone.** The provider never touches a workspace's state (versions, rollbacks, locks), and
  deleting a workspace whose state still manages resources fails: destroy them in Opsolete first.
- **Sensitive values are write-only** in Opsolete (variables, VCS tokens): Terraform keeps the configured value and
  can't see a change made to it in the UI. They're in Terraform's state: keep that encrypted.
- **Changes in the UI to managed objects** are reverted by the next apply, as with any provider.

## Development

```sh
go build ./... && go vet ./...
go generate ./...   # docs/ from the schema and examples/
```

Acceptance tests run real Terraform against a running Opsolete, creating and deleting objects with random names:

```sh
TF_ACC=1 OPSOLETE_URL=http://localhost:8080 OPSOLETE_TOKEN=<service token> go test -v ./...
# optional: a template's per-environment input, and a GitHub token Opsolete can check
OPSOLETE_ACC_TEMPLATE_ID=<id> OPSOLETE_ACC_INPUT=location OPSOLETE_ACC_VALUES=westeurope,northeurope
OPSOLETE_ACC_GITHUB_TOKEN=<token>
```

CI (`.github/workflows/test.yml`) builds Opsolete's `master`, starts it on Postgres and runs them. A tag `v*` releases
(`.github/workflows/release.yml`, goreleaser): it needs the secrets `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE`, and the
provider added to the Terraform Registry with that key's public half.
