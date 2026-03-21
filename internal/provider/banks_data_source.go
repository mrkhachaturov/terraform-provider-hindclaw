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
	_ datasource.DataSource              = &banksDataSource{}
	_ datasource.DataSourceWithConfigure = &banksDataSource{}
)

func NewBanksDataSource() datasource.DataSource {
	return &banksDataSource{}
}

type banksDataSourceModel struct {
	Banks []bankItemModel `tfsdk:"banks"`
}

type bankItemModel struct {
	BankID types.String `tfsdk:"bank_id"`
	Name   types.String `tfsdk:"name"`
}

type banksDataSource struct {
	client *hindsight.APIClient
}

func (d *banksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_banks"
}

func (d *banksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Hindsight banks.",
		Attributes: map[string]schema.Attribute{
			"banks": schema.ListNestedAttribute{
				Description: "List of banks.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"bank_id": schema.StringAttribute{Description: "Bank ID.", Computed: true},
						"name":    schema.StringAttribute{Description: "Bank name.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *banksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *banksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	bankList, _, err := d.client.BanksAPI.ListBanks(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error listing banks", err.Error())
		return
	}

	var state banksDataSourceModel
	for _, b := range bankList.Banks {
		item := bankItemModel{
			BankID: types.StringValue(b.BankId),
		}
		if b.Name.IsSet() {
			item.Name = types.StringValue(*b.Name.Get())
		} else {
			item.Name = types.StringNull()
		}
		state.Banks = append(state.Banks, item)
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
