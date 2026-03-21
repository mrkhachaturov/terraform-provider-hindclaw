package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	hindclaw "github.com/mrkhachaturov/hindclaw/hindclaw-clients/go"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &strategyScopeResource{}
	_ resource.ResourceWithConfigure   = &strategyScopeResource{}
	_ resource.ResourceWithImportState = &strategyScopeResource{}
)

func NewStrategyScopeResource() resource.Resource {
	return &strategyScopeResource{}
}

type strategyScopeResourceModel struct {
	BankID     types.String `tfsdk:"bank_id"`
	ScopeType  types.String `tfsdk:"scope_type"`
	ScopeValue types.String `tfsdk:"scope_value"`
	Strategy   types.String `tfsdk:"strategy"`
}

type strategyScopeResource struct {
	client *hindclaw.APIClient
}

func (r *strategyScopeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_strategy_scope"
}

func (r *strategyScopeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a retain strategy binding for a bank scope.",
		Attributes: map[string]schema.Attribute{
			"bank_id": schema.StringAttribute{
				Description: "Bank identifier.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scope_type": schema.StringAttribute{
				Description: "Scope type (e.g. topic, channel).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scope_value": schema.StringAttribute{
				Description: "Scope value (e.g. topic ID).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"strategy": schema.StringAttribute{
				Description: "Retain strategy name.",
				Required:    true,
			},
		},
	}
}

func (r *strategyScopeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = clients.hindclaw
}

func (r *strategyScopeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan strategyScopeResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stratReq := hindclaw.NewStrategyRequest(plan.Strategy.ValueString())

	_, _, err := r.client.DefaultAPI.UpsertStrategy(ctx,
		plan.BankID.ValueString(),
		plan.ScopeType.ValueString(),
		plan.ScopeValue.ValueString(),
	).StrategyRequest(*stratReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating strategy scope", err.Error())
		return
	}

	tflog.Trace(ctx, "created strategy scope", map[string]any{
		"bank_id":     plan.BankID.ValueString(),
		"scope_type":  plan.ScopeType.ValueString(),
		"scope_value": plan.ScopeValue.ValueString(),
	})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *strategyScopeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state strategyScopeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	strategies, httpResp, err := r.client.DefaultAPI.ListStrategies(ctx, state.BankID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading strategies", err.Error())
		return
	}

	found := false
	for _, s := range strategies {
		if s.ScopeType == state.ScopeType.ValueString() && s.ScopeValue == state.ScopeValue.ValueString() {
			state.Strategy = types.StringValue(s.Strategy)
			found = true
			break
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *strategyScopeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan strategyScopeResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stratReq := hindclaw.NewStrategyRequest(plan.Strategy.ValueString())

	_, _, err := r.client.DefaultAPI.UpsertStrategy(ctx,
		plan.BankID.ValueString(),
		plan.ScopeType.ValueString(),
		plan.ScopeValue.ValueString(),
	).StrategyRequest(*stratReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating strategy scope", err.Error())
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *strategyScopeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state strategyScopeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.DefaultAPI.DeleteStrategy(ctx,
		state.BankID.ValueString(),
		state.ScopeType.ValueString(),
		state.ScopeValue.ValueString(),
	).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting strategy scope", err.Error())
		return
	}
}

func (r *strategyScopeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: {bank_id}/{scope_type}/{scope_value}")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bank_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope_type"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope_value"), parts[2])...)
}
