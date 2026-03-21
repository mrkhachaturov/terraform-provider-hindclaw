package provider

import (
	"context"
	"os"
	"time"

	hindclaw "github.com/mrkhachaturov/hindclaw/hindclaw-clients/go"
	hindsight "github.com/vectorize-io/hindsight/hindsight-clients/go"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ provider.Provider = &hindclawProvider{}
)

// hindclawClients holds both API clients. Passed to resources via ProviderData.
type hindclawClients struct {
	hindclaw  *hindclaw.APIClient
	hindsight *hindsight.APIClient
}

// hindclawProviderModel maps provider schema to Go types.
type hindclawProviderModel struct {
	ApiUrl types.String `tfsdk:"api_url"`
	ApiKey types.String `tfsdk:"api_key"`
}

type hindclawProvider struct {
	version string
}

// New is the provider factory called from main.go.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &hindclawProvider{version: version}
	}
}

func (p *hindclawProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hindclaw"
	resp.Version = p.version
}

func (p *hindclawProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Hindsight memory banks and Hindclaw access control.",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Description: "Hindsight API server URL. May also be provided via HINDCLAW_API_URL.",
				Optional:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "API key for authentication. May also be provided via HINDCLAW_API_KEY.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *hindclawProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Hindclaw provider")

	var config hindclawProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate unknown values
	if config.ApiUrl.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("api_url"),
			"Unknown Hindclaw API URL",
			"Set the api_url value statically or use the HINDCLAW_API_URL environment variable.")
	}
	if config.ApiKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"),
			"Unknown Hindclaw API Key",
			"Set the api_key value statically or use the HINDCLAW_API_KEY environment variable.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Env var fallback
	apiUrl := os.Getenv("HINDCLAW_API_URL")
	apiKey := os.Getenv("HINDCLAW_API_KEY")
	if !config.ApiUrl.IsNull() {
		apiUrl = config.ApiUrl.ValueString()
	}
	if !config.ApiKey.IsNull() {
		apiKey = config.ApiKey.ValueString()
	}

	if apiUrl == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_url"),
			"Missing Hindclaw API URL",
			"Set api_url in provider config or HINDCLAW_API_URL env var.")
	}
	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"),
			"Missing Hindclaw API Key",
			"Set api_key in provider config or HINDCLAW_API_KEY env var.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "hindclaw_api_url", apiUrl)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "hindclaw_api_key")
	tflog.Debug(ctx, "Creating Hindclaw clients")

	clients := &hindclawClients{
		hindclaw:  hindclaw.NewAPIClientWithTimeout(apiUrl, apiKey, 30*time.Second),
		hindsight: hindsight.NewAPIClientWithTimeout(apiUrl, apiKey, 30*time.Second),
	}

	resp.DataSourceData = clients
	resp.ResourceData = clients

	tflog.Info(ctx, "Configured Hindclaw provider", map[string]any{"success": true})
}

func (p *hindclawProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
		NewGroupResource,
		NewUserChannelResource,
		NewGroupMembershipResource,
		NewBankPermissionResource,
		NewStrategyScopeResource,
		NewApiKeyResource,
		NewBankResource,
		NewBankConfigResource,
		NewMentalModelResource,
		NewDirectiveResource,
		NewWebhookResource,
	}
}

func (p *hindclawProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewResolvedPermissionsDataSource,
		NewBankProfileDataSource,
		NewBanksDataSource,
	}
}
