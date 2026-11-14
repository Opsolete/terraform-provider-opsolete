package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/opsolete/terraform-provider-opsolete/internal/client"
)

// Acceptance tests run real Terraform against a real Opsolete (TF_ACC=1, OPSOLETE_URL, OPSOLETE_TOKEN: a service
// token). Every object they create has a random suffix and is destroyed at the end.

var factories = map[string]func() (tfprotov6.ProviderServer, error){
	"opsolete": providerserver.NewProtocol6WithError(New("test")()),
}

func preCheck(t *testing.T) {
	t.Helper()
	for _, k := range []string{"OPSOLETE_URL", "OPSOLETE_TOKEN"} {
		if os.Getenv(k) == "" {
			t.Fatalf("%s is not set: acceptance tests need a running Opsolete and a service token", k)
		}
	}
}

func suffix() string { return fmt.Sprintf("%x", time.Now().UnixNano())[8:] }

// TestAccProject: a project is created, changed in place, imported by name, and read by a data source.
func TestAccProject(t *testing.T) {
	name := "acc-" + suffix()
	cfg := func(desc string) string {
		return fmt.Sprintf(`
resource "opsolete_project" "p" {
  name        = %q
  description = %q
  runner_pool = "acc"
}
data "opsolete_project" "p" {
  name       = opsolete_project.p.name
  depends_on = [opsolete_project.p]
}`, name, desc)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: cfg("first"), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("opsolete_project.p", "description", "first"),
				resource.TestCheckResourceAttr("opsolete_project.p", "runner_pool", "acc"),
				resource.TestCheckResourceAttr("opsolete_project.p", "default", "false"),
				resource.TestCheckResourceAttrPair("data.opsolete_project.p", "id", "opsolete_project.p", "id"))},
			{Config: cfg("second"), Check: resource.TestCheckResourceAttr("opsolete_project.p", "description", "second")},
			{ResourceName: "opsolete_project.p", ImportState: true, ImportStateId: name, ImportStateVerify: true},
		},
	})
}

// TestAccWorkspace: a project, an empty workspace in it with its settings and variables (a sensitive one write-only),
// the workspace renamed in place (same id), imported back.
func TestAccWorkspace(t *testing.T) {
	sfx := suffix()
	cfg := func(ws, plain string) string {
		return fmt.Sprintf(`
resource "opsolete_project" "p" { name = "acc-%[1]s" }
resource "opsolete_workspace" "w" {
  name            = %[2]q
  project_id      = opsolete_project.p.id
  drift_detection = true
}
resource "opsolete_workspace_variable" "plain" {
  workspace_id = opsolete_workspace.w.id
  category     = "terraform"
  key          = "region"
  value        = %[3]q
}
resource "opsolete_workspace_variable" "secret" {
  workspace_id = opsolete_workspace.w.id
  category     = "env"
  key          = "ARM_CLIENT_SECRET"
  value        = "s3cret"
  sensitive    = true
}
resource "opsolete_workspace_variable" "tags" {
  workspace_id = opsolete_workspace.w.id
  category     = "terraform"
  key          = "tags"
  value        = "{env = \"acc\"}"
  hcl          = true
}`, sfx, ws, plain)
	}
	var firstID string
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: cfg("acc-ws-"+sfx, "westeurope"), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair("opsolete_workspace.w", "project_id", "opsolete_project.p", "id"),
				resource.TestCheckResourceAttr("opsolete_workspace.w", "execution_mode", "remote"),
				resource.TestCheckResourceAttr("opsolete_workspace.w", "drift_detection", "true"),
				resource.TestCheckResourceAttr("opsolete_workspace_variable.secret", "value", "s3cret"), // kept: not returned
				func(s *terraform.State) error {
					firstID = s.RootModule().Resources["opsolete_workspace.w"].Primary.ID
					return nil
				})},
			{Config: cfg("acc-ws2-"+sfx, "northeurope"), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("opsolete_workspace.w", "name", "acc-ws2-"+sfx),
				resource.TestCheckResourceAttr("opsolete_workspace_variable.plain", "value", "northeurope"),
				func(s *terraform.State) error {
					if id := s.RootModule().Resources["opsolete_workspace.w"].Primary.ID; id != firstID {
						return fmt.Errorf("renaming replaced the workspace: %s → %s", firstID, id)
					}
					return nil
				})},
			{ResourceName: "opsolete_workspace.w", ImportState: true, ImportStateId: "acc-ws2-" + sfx, ImportStateVerify: true},
			{ResourceName: "opsolete_workspace_variable.plain", ImportState: true, ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r := s.RootModule().Resources["opsolete_workspace_variable.plain"].Primary
					return r.Attributes["workspace_id"] + "/" + r.ID, nil
				}},
		},
	})
}

// TestAccVariableSet: a set with a variable, attached to a project and a workspace, imported back.
func TestAccVariableSet(t *testing.T) {
	sfx := suffix()
	cfg := fmt.Sprintf(`
resource "opsolete_project" "p" { name = "acc-%[1]s" }
resource "opsolete_workspace" "w" {
  name       = "acc-ws-%[1]s"
  project_id = opsolete_project.p.id
}
resource "opsolete_variable_set" "s" {
  name     = "acc-set-%[1]s"
  priority = true
}
resource "opsolete_variable_set_variable" "v" {
  variable_set_id = opsolete_variable_set.s.id
  category        = "env"
  key             = "ARM_SUBSCRIPTION_ID"
  value           = "00000000-0000-0000-0000-000000000001"
}
resource "opsolete_variable_set_project" "p" {
  variable_set_id = opsolete_variable_set.s.id
  project_id      = opsolete_project.p.id
}
resource "opsolete_variable_set_workspace" "w" {
  variable_set_id = opsolete_variable_set.s.id
  workspace_id    = opsolete_workspace.w.id
}
data "opsolete_variable_set" "s" {
  name       = opsolete_variable_set.s.name
  depends_on = [opsolete_variable_set.s]
}`, sfx)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: cfg, Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("opsolete_variable_set.s", "priority", "true"),
				resource.TestCheckResourceAttrPair("data.opsolete_variable_set.s", "id", "opsolete_variable_set.s", "id"))},
			{ResourceName: "opsolete_variable_set.s", ImportState: true, ImportStateId: "acc-set-" + sfx, ImportStateVerify: true},
			{ResourceName: "opsolete_variable_set_project.p", ImportState: true, ImportStateVerify: true},
			{ResourceName: "opsolete_variable_set_workspace.w", ImportState: true, ImportStateVerify: true},
		},
	})
}

// TestAccEnvironmentValue sets a project's value for a template's per-environment input: needs such a template in the
// catalog (OPSOLETE_ACC_TEMPLATE_ID, OPSOLETE_ACC_INPUT, OPSOLETE_ACC_VALUES: two allowed values, comma-separated).
func TestAccEnvironmentValue(t *testing.T) {
	tpl, input, values := os.Getenv("OPSOLETE_ACC_TEMPLATE_ID"), os.Getenv("OPSOLETE_ACC_INPUT"), strings.Split(os.Getenv("OPSOLETE_ACC_VALUES"), ",")
	if tpl == "" || input == "" || len(values) != 2 {
		t.Skip("OPSOLETE_ACC_TEMPLATE_ID, OPSOLETE_ACC_INPUT and OPSOLETE_ACC_VALUES name a template's per-environment input")
	}
	sfx := suffix()
	cfg := func(value string) string {
		return fmt.Sprintf(`
resource "opsolete_project" "p" { name = "acc-%s" }
resource "opsolete_environment_value" "v" {
  project_id  = opsolete_project.p.id
  template_id = %q
  input       = %q
  value       = %q
}`, sfx, tpl, input, value)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: cfg(values[0]), Check: resource.TestCheckResourceAttr("opsolete_environment_value.v", "value", values[0])},
			{Config: cfg(values[1]), Check: resource.TestCheckResourceAttr("opsolete_environment_value.v", "value", values[1])},
			{ResourceName: "opsolete_environment_value.v", ImportState: true, ImportStateVerify: true},
		},
	})
}

// TestAccVCSConnection creates a GitHub connection: needs a token GitHub accepts (OPSOLETE_ACC_GITHUB_TOKEN), as
// Opsolete checks it.
func TestAccVCSConnection(t *testing.T) {
	token := os.Getenv("OPSOLETE_ACC_GITHUB_TOKEN")
	if token == "" {
		t.Skip("OPSOLETE_ACC_GITHUB_TOKEN: a GitHub token Opsolete can check")
	}
	sfx := suffix()
	cfg := func(scope string) string {
		return fmt.Sprintf(`
variable "token" {
  type      = string
  sensitive = true
}
resource "opsolete_vcs_connection" "c" {
  name          = "acc-%s"
  provider_type = "github"
  token         = var.token
  scope         = %q
}`, sfx, scope)
	}
	t.Setenv("TF_VAR_token", token)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: cfg(""), Check: resource.TestCheckResourceAttr("opsolete_vcs_connection.c", "provider_type", "github")},
			{Config: cfg("all"), Check: resource.TestCheckResourceAttr("opsolete_vcs_connection.c", "scope", "all")},
			{ResourceName: "opsolete_vcs_connection.c", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"token"}},
		},
	})
}

// TestAccRunner: a runner with a token Opsolete creates (kept in the state), and one with a token from a secret store.
func TestAccRunner(t *testing.T) {
	sfx := suffix()
	cfg := fmt.Sprintf(`
resource "opsolete_runner" "made" {
  name = "acc-%[1]s"
  pool = "acc"
}
resource "opsolete_runner" "own" {
  name  = "acc-own-%[1]s"
  token = %[2]q
}`, sfx, strings.Repeat("t", 40))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: cfg, Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("opsolete_runner.made", "pool", "acc"),
				resource.TestCheckResourceAttrWith("opsolete_runner.made", "token", func(v string) error {
					if !strings.HasPrefix(v, "opsr_") {
						return fmt.Errorf("a created runner's token: %q", v)
					}
					return nil
				}),
				resource.TestCheckResourceAttr("opsolete_runner.own", "pool", "default"))},
			{ResourceName: "opsolete_runner.made", ImportState: true, ImportStateId: "acc-" + sfx, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"token"}},
		},
	})
}

// TestAccRegistryModule publishes a module from a public GitHub repository's tags: needs a GitHub token Opsolete can
// check (OPSOLETE_ACC_GITHUB_TOKEN); Opsolete/terraform-modules-examples' storage-account by default
// (OPSOLETE_ACC_MODULE_REPO, OPSOLETE_ACC_MODULE_DIR: its folder and tag prefix <dir>-).
func TestAccRegistryModule(t *testing.T) {
	token := os.Getenv("OPSOLETE_ACC_GITHUB_TOKEN")
	if token == "" {
		t.Skip("OPSOLETE_ACC_GITHUB_TOKEN: a GitHub token Opsolete can check")
	}
	repo, dir := envOr("OPSOLETE_ACC_MODULE_REPO", "Opsolete/terraform-modules-examples"), envOr("OPSOLETE_ACC_MODULE_DIR", "storage-account")
	sfx := suffix()
	t.Setenv("TF_VAR_token", token)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: fmt.Sprintf(`
variable "token" {
  type      = string
  sensitive = true
}
resource "opsolete_vcs_connection" "gh" {
  name          = "acc-%[1]s"
  provider_type = "github"
  token         = var.token
}
data "opsolete_vcs_connection" "gh" {
  name       = opsolete_vcs_connection.gh.name
  depends_on = [opsolete_vcs_connection.gh]
}
resource "opsolete_registry_module" "m" {
  vcs_connection_id = data.opsolete_vcs_connection.gh.id
  repo              = %[2]q
  dir               = %[3]q
  tag_prefix        = "%[3]s-"
  namespace         = "acc%[1]s"
  name              = %[3]q
  system            = "azurerm"
}`, sfx, repo, dir), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("opsolete_registry_module.m", "id", "acc"+sfx+"/"+dir+"/azurerm"),
				resource.TestCheckResourceAttrWith("opsolete_registry_module.m", "versions.#", func(v string) error {
					if v == "0" {
						return fmt.Errorf("no version published from the tags")
					}
					return nil
				}))},
			{ResourceName: "opsolete_registry_module.m", ImportState: true, ImportStateVerify: true},
		},
	})
}

// TestAccTemplateCatalog connects the template catalog to a GitHub repository (OPSOLETE_ACC_GITHUB_TOKEN; the repo of
// TestAccRegistryModule, a folder without templates) and disconnects it. Destroying it disconnects the catalog, so it
// refuses to run on an Opsolete whose catalog is connected already.
func TestAccTemplateCatalog(t *testing.T) {
	token := os.Getenv("OPSOLETE_ACC_GITHUB_TOKEN")
	if token == "" {
		t.Skip("OPSOLETE_ACC_GITHUB_TOKEN: a GitHub token Opsolete can check")
	}
	preCheck(t)
	c := testClient(t)
	if err := c.Do(context.Background(), "GET", "/template-catalog", nil, nil); err == nil {
		t.Skip("this Opsolete's template catalog is connected: the test would disconnect it")
	}
	repo, dir := envOr("OPSOLETE_ACC_MODULE_REPO", "Opsolete/terraform-modules-examples"), envOr("OPSOLETE_ACC_MODULE_DIR", "storage-account")
	t.Setenv("TF_VAR_token", token)
	cfg := func(mode string) string {
		return fmt.Sprintf(`
variable "token" {
  type      = string
  sensitive = true
}
resource "opsolete_vcs_connection" "gh" {
  name          = "acc-cat-%[1]s"
  provider_type = "github"
  token         = var.token
}
resource "opsolete_template_catalog" "c" {
  vcs_connection_id = opsolete_vcs_connection.gh.id
  repo              = %[2]q
  dir               = %[3]q
  mode              = %[4]q
}`, suffixOnce, repo, dir, mode)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() {},
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: cfg("off"), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("opsolete_template_catalog.c", "branch", "main"),
				resource.TestCheckResourceAttrSet("opsolete_template_catalog.c", "commit"))},
			{Config: cfg("pr"), Check: resource.TestCheckResourceAttr("opsolete_template_catalog.c", "mode", "pr")},
			{ResourceName: "opsolete_template_catalog.c", ImportState: true, ImportStateVerify: true},
		},
	})
}

var suffixOnce = suffix()

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func testClient(t *testing.T) *client.Client {
	t.Helper()
	c, err := client.New(os.Getenv("OPSOLETE_URL"), os.Getenv("OPSOLETE_TOKEN"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}
