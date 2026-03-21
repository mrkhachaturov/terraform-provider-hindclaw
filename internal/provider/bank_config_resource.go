package provider

import (
	"context"
	"fmt"
	"net/http"

	hindsight "github.com/vectorize-io/hindsight/hindsight-clients/go"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &bankConfigResource{}
	_ resource.ResourceWithConfigure   = &bankConfigResource{}
	_ resource.ResourceWithImportState = &bankConfigResource{}
)

func NewBankConfigResource() resource.Resource {
	return &bankConfigResource{}
}

type bankConfigResourceModel struct {
	BankID types.String `tfsdk:"bank_id"`
	Config types.Map    `tfsdk:"config"`
}

type bankConfigResource struct {
	client *hindsight.APIClient
}

func (r *bankConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bank_config"
}

func (r *bankConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages bank configuration overrides. Delete resets to defaults (bank remains).",
		Attributes: map[string]schema.Attribute{
			"bank_id": schema.StringAttribute{
				Description: "Bank identifier.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config": schema.MapAttribute{
				Description: "Configuration key-value overrides.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *bankConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*hindclawClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *hindclawClients, got: %T", req.ProviderData),
		)
		return
	}
	r.client = clients.hindsight
}

func (r *bankConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bankConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configReq, err := r.buildConfigUpdate(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err = r.client.BanksAPI.UpdateBankConfig(ctx, plan.BankID.ValueString()).BankConfigUpdate(*configReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating bank config", err.Error())
		return
	}

	if notFound := r.readOverridesIntoState(ctx, plan.BankID.ValueString(), &plan, &resp.Diagnostics); notFound {
		resp.Diagnostics.AddError("Error reading bank config after create", "Bank not found")
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created bank config", map[string]any{"bank_id": plan.BankID.ValueString()})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *bankConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bankConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if notFound := r.readOverridesIntoState(ctx, state.BankID.ValueString(), &state, &resp.Diagnostics); notFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *bankConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bankConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// UpdateBankConfig is PATCH (merge), so removing a key from Terraform config
	// won't clear it on the server. Reset first, then re-apply desired overrides.
	_, _, err := r.client.BanksAPI.ResetBankConfig(ctx, plan.BankID.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error resetting bank config before update", err.Error())
		return
	}

	configReq, err := r.buildConfigUpdate(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err = r.client.BanksAPI.UpdateBankConfig(ctx, plan.BankID.ValueString()).BankConfigUpdate(*configReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating bank config", err.Error())
		return
	}

	if notFound := r.readOverridesIntoState(ctx, plan.BankID.ValueString(), &plan, &resp.Diagnostics); notFound {
		resp.Diagnostics.AddError("Error reading bank config after update", "Bank not found")
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *bankConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bankConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, httpResp, err := r.client.BanksAPI.ResetBankConfig(ctx, state.BankID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error resetting bank config", err.Error())
		return
	}
}

func (r *bankConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bank_id"), req, resp)
}

func (r *bankConfigResource) buildConfigUpdate(ctx context.Context, plan *bankConfigResourceModel, diags *diag.Diagnostics) (*hindsight.BankConfigUpdate, error) {
	updates := make(map[string]interface{})
	var configMap map[string]string
	d := plan.Config.ElementsAs(ctx, &configMap, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to parse config map")
	}
	for k, v := range configMap {
		updates[k] = v
	}
	return hindsight.NewBankConfigUpdate(updates), nil
}

// readOverridesIntoState reads the Overrides map (not resolved Config) from the API.
// Returns true if 404 (bank deleted).
func (r *bankConfigResource) readOverridesIntoState(ctx context.Context, bankID string, state *bankConfigResourceModel, diags *diag.Diagnostics) (notFound bool) {
	bankConfig, httpResp, err := r.client.BanksAPI.GetBankConfig(ctx, bankID).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return true
		}
		diags.AddError("Error reading bank config", err.Error())
		return false
	}

	// Store only Overrides, not the resolved Config (which includes inherited defaults).
	overrides := make(map[string]string)
	for k, v := range bankConfig.Overrides {
		overrides[k] = fmt.Sprintf("%v", v)
	}

	mapVal, d := types.MapValueFrom(ctx, types.StringType, overrides)
	diags.Append(d...)
	state.Config = mapVal
	return false
}
