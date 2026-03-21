package provider

import (
	"context"
	"fmt"

	hindsight "github.com/vectorize-io/hindsight/hindsight-clients/go"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &bankProfileDataSource{}
	_ datasource.DataSourceWithConfigure = &bankProfileDataSource{}
)

func NewBankProfileDataSource() datasource.DataSource {
	return &bankProfileDataSource{}
}

type bankProfileDataSourceModel struct {
	BankID                types.String `tfsdk:"bank_id"`
	Name                  types.String `tfsdk:"name"`
	Mission               types.String `tfsdk:"mission"`
	Background            types.String `tfsdk:"background"`
	DispositionSkepticism types.Int64  `tfsdk:"disposition_skepticism"`
	DispositionLiteralism types.Int64  `tfsdk:"disposition_literalism"`
	DispositionEmpathy    types.Int64  `tfsdk:"disposition_empathy"`
}

type bankProfileDataSource struct {
	client *hindsight.APIClient
}

func (d *bankProfileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bank_profile"
}

func (d *bankProfileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Hindsight bank profile.",
		Attributes: map[string]schema.Attribute{
			"bank_id":                schema.StringAttribute{Description: "Bank ID.", Required: true},
			"name":                   schema.StringAttribute{Description: "Bank name.", Computed: true},
			"mission":                schema.StringAttribute{Description: "Bank mission.", Computed: true},
			"background":             schema.StringAttribute{Description: "Background context.", Computed: true},
			"disposition_skepticism": schema.Int64Attribute{Description: "Skepticism (1-5).", Computed: true},
			"disposition_literalism": schema.Int64Attribute{Description: "Literalism (1-5).", Computed: true},
			"disposition_empathy":    schema.Int64Attribute{Description: "Empathy (1-5).", Computed: true},
		},
	}
}

func (d *bankProfileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*hindclawClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *hindclawClients, got: %T", req.ProviderData),
		)
		return
	}
	d.client = clients.hindsight
}

func (d *bankProfileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config bankProfileDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	profile, _, err := d.client.BanksAPI.GetBankProfile(ctx, config.BankID.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error reading bank profile", err.Error())
		return
	}

	config.Name = types.StringValue(profile.Name)
	config.Mission = types.StringValue(profile.Mission)
	if profile.Background.IsSet() {
		config.Background = types.StringValue(*profile.Background.Get())
	} else {
		config.Background = types.StringNull()
	}
	config.DispositionSkepticism = types.Int64Value(int64(profile.Disposition.Skepticism))
	config.DispositionLiteralism = types.Int64Value(int64(profile.Disposition.Literalism))
	config.DispositionEmpathy = types.Int64Value(int64(profile.Disposition.Empathy))

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
