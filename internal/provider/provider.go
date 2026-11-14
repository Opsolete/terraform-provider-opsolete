// Package provider is the Opsolete Terraform provider: Opsolete's platform objects (projects and their environment
// values, workspaces and their variables, variable sets, VCS connections) through its JSON API.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/opsolete/terraform-provider-opsolete/internal/client"
)

// New returns the provider of version (set at release).
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &opsoleteProvider{version: version} }
}

type opsoleteProvider struct{ version string }

type providerModel struct {
	URL   types.String `tfsdk:"url"`
	Token types.String `tfsdk:"token"`
}

func (p *opsoleteProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName, resp.Version = "opsolete", p.version
}

func (p *opsoleteProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Opsolete's platform objects: projects and their environment values, workspaces and their variables, variable sets and VCS connections. Templates on workspaces, runs and team access stay in Opsolete's UI.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "Opsolete's address, e.g. `https://opsolete.example.com`. Default: the `OPSOLETE_URL` environment variable.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "A service token (Settings → API tokens, or `opsolete api-token <name>` on the server): admin rights, audited as `token:<name>`. Default: the `OPSOLETE_TOKEN` environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *opsoleteProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var m providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	url, token := m.URL.ValueString(), m.Token.ValueString()
	if url == "" {
		url = os.Getenv("OPSOLETE_URL")
	}
	if token == "" {
		token = os.Getenv("OPSOLETE_TOKEN")
	}
	if url == "" || token == "" {
		resp.Diagnostics.AddError("Opsolete isn't configured",
			"Set the provider's url and token, or the OPSOLETE_URL and OPSOLETE_TOKEN environment variables. Create a token under Settings → API tokens.")
		return
	}
	c, err := client.New(url, token)
	if err != nil {
		resp.Diagnostics.AddAttributeError(pathRoot("url"), "Invalid url", err.Error())
		return
	}
	resp.ResourceData, resp.DataSourceData = c, c
}

func (p *opsoleteProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newProjectResource, newEnvironmentValueResource, newWorkspaceResource, newWorkspaceVariableResource,
		newVariableSetResource, newVariableSetVariableResource, newVariableSetProjectResource, newVariableSetWorkspaceResource,
		newVCSConnectionResource,
	}
}

func (p *opsoleteProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{newProjectDataSource, newWorkspaceDataSource, newVariableSetDataSource}
}
