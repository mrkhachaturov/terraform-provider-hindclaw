package provider

import (
	"context"
	"encoding/json"
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
	Config types.String `tfsdk:"config"`
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
			"config": schema.StringAttribute{
				Description: "Configuration overrides as JSON object. Use jsonencode() to set typed values.",
				Required:    true,
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

	if notFound := r.readManagedOverridesIntoState(ctx, plan.BankID.ValueString(), plan.Config.ValueString(), &plan, &resp.Diagnostics); notFound {
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

	if notFound := r.readManagedOverridesIntoState(ctx, state.BankID.ValueString(), state.Config.ValueString(), &state, &resp.Diagnostics); notFound {
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
	var state bankConfigResourceModel
	var plan bankConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	desiredOverrides, err := r.parseConfigJSON(plan.Config.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	currentConfig, httpResp, err := r.client.BanksAPI.GetBankConfig(ctx, plan.BankID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddError("Error reading bank config before update", "Bank not found")
			return
		}
		resp.Diagnostics.AddError("Error reading bank config before update", err.Error())
		return
	}

	managedKeys, err := parseConfigKeys(state.Config.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid prior state config JSON", err.Error())
		return
	}

	// UpdateBankConfig is PATCH (merge). To remove managed keys safely without
	// clobbering overrides owned by other resources, reset the bank config and
	// then re-apply:
	// 1. all current overrides not owned by this resource
	// 2. the desired overrides from Terraform config
	mergedOverrides := filterOverridesExcludingKeys(currentConfig.Overrides, managedKeys)
	for k, v := range desiredOverrides {
		mergedOverrides[k] = v
	}

	_, _, err = r.client.BanksAPI.ResetBankConfig(ctx, plan.BankID.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error resetting bank config before update", err.Error())
		return
	}

	if len(mergedOverrides) > 0 {
		configReq := hindsight.NewBankConfigUpdate(mergedOverrides)
		_, _, err = r.client.BanksAPI.UpdateBankConfig(ctx, plan.BankID.ValueString()).BankConfigUpdate(*configReq).Execute()
		if err != nil {
			resp.Diagnostics.AddError("Error updating bank config", err.Error())
			return
		}
	}

	if notFound := r.readManagedOverridesIntoState(ctx, plan.BankID.ValueString(), plan.Config.ValueString(), &plan, &resp.Diagnostics); notFound {
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

	currentConfig, httpResp, err := r.client.BanksAPI.GetBankConfig(ctx, state.BankID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error reading bank config before delete", err.Error())
		return
	}

	managedKeys, err := parseConfigKeys(state.Config.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid prior state config JSON", err.Error())
		return
	}

	preservedOverrides := filterOverridesExcludingKeys(currentConfig.Overrides, managedKeys)

	_, httpResp, err = r.client.BanksAPI.ResetBankConfig(ctx, state.BankID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error resetting bank config", err.Error())
		return
	}

	if len(preservedOverrides) > 0 {
		configReq := hindsight.NewBankConfigUpdate(preservedOverrides)
		_, httpResp, err = r.client.BanksAPI.UpdateBankConfig(ctx, state.BankID.ValueString()).BankConfigUpdate(*configReq).Execute()
		if err != nil {
			if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
				return
			}
			resp.Diagnostics.AddError("Error restoring unmanaged bank config overrides", err.Error())
			return
		}
	}
}

func (r *bankConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bank_id"), req, resp)
}

func (r *bankConfigResource) buildConfigUpdate(ctx context.Context, plan *bankConfigResourceModel, diags *diag.Diagnostics) (*hindsight.BankConfigUpdate, error) {
	updates, err := r.parseConfigJSON(plan.Config.ValueString(), diags)
	if err != nil {
		return nil, err
	}
	return hindsight.NewBankConfigUpdate(updates), nil
}

func (r *bankConfigResource) parseConfigJSON(configJSON string, diags *diag.Diagnostics) (map[string]interface{}, error) {
	var updates map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &updates); err != nil {
		diags.AddError("Invalid config JSON", err.Error())
		return nil, err
	}
	return updates, nil
}

func parseConfigKeys(configJSON string) (map[string]struct{}, error) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}
	keys := make(map[string]struct{}, len(config))
	for key := range config {
		keys[key] = struct{}{}
	}
	return keys, nil
}

func filterOverridesExcludingKeys(overrides map[string]interface{}, excludedKeys map[string]struct{}) map[string]interface{} {
	filtered := make(map[string]interface{}, len(overrides))
	for key, value := range overrides {
		if _, excluded := excludedKeys[key]; excluded {
			continue
		}
		filtered[key] = value
	}
	return filtered
}

// readManagedOverridesIntoState reads the bank-specific Overrides map from the API,
// then filters it down to only the keys managed by this Terraform resource.
// This avoids state drift when the API also returns inherited or foreign overrides
// in the same namespace. Returns true if 404 (bank deleted).
func (r *bankConfigResource) readManagedOverridesIntoState(ctx context.Context, bankID string, managedConfigJSON string, state *bankConfigResourceModel, diags *diag.Diagnostics) (notFound bool) {
	bankConfig, httpResp, err := r.client.BanksAPI.GetBankConfig(ctx, bankID).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return true
		}
		diags.AddError("Error reading bank config", err.Error())
		return false
	}

	managedKeys, err := parseConfigKeys(managedConfigJSON)
	if err != nil {
		diags.AddError("Invalid managed config JSON in state", err.Error())
		return false
	}

	managedOverrides := make(map[string]interface{}, len(managedKeys))
	for key := range managedKeys {
		if value, ok := bankConfig.Overrides[key]; ok {
			managedOverrides[key] = value
		}
	}

	jsonBytes, err := json.Marshal(managedOverrides)
	if err != nil {
		diags.AddError("Error marshaling managed overrides", err.Error())
		return false
	}
	state.Config = types.StringValue(string(jsonBytes))
	return false
}
