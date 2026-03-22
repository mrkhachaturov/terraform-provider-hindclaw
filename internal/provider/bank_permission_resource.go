package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	hindclaw "github.com/mrkhachaturov/hindclaw/hindclaw-clients/go"

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
	_ resource.Resource                = &bankPermissionResource{}
	_ resource.ResourceWithConfigure   = &bankPermissionResource{}
	_ resource.ResourceWithImportState = &bankPermissionResource{}
)

func NewBankPermissionResource() resource.Resource {
	return &bankPermissionResource{}
}

type bankPermissionResourceModel struct {
	BankID            types.String `tfsdk:"bank_id"`
	ScopeType         types.String `tfsdk:"scope_type"`
	ScopeID           types.String `tfsdk:"scope_id"`
	Recall            types.Bool   `tfsdk:"recall"`
	Retain            types.Bool   `tfsdk:"retain"`
	RetainTags        types.List   `tfsdk:"retain_tags"`
	RetainRoles       types.List   `tfsdk:"retain_roles"`
	RetainEveryNTurns types.Int64  `tfsdk:"retain_every_n_turns"`
	RecallBudget      types.String `tfsdk:"recall_budget"`
	RecallMaxTokens   types.Int64  `tfsdk:"recall_max_tokens"`
	RecallTagGroups   types.String `tfsdk:"recall_tag_groups"`
	LlmModel          types.String `tfsdk:"llm_model"`
	LlmProvider       types.String `tfsdk:"llm_provider"`
	ExcludeProviders  types.List   `tfsdk:"exclude_providers"`
	RetainStrategy    types.String `tfsdk:"retain_strategy"`
}

type bankPermissionResource struct {
	client *hindclaw.APIClient
}

func (r *bankPermissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bank_permission"
}

func (r *bankPermissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages per-bank permission overrides for a user or group.",
		Attributes: map[string]schema.Attribute{
			"bank_id": schema.StringAttribute{
				Description: "Bank identifier.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scope_type": schema.StringAttribute{
				Description: "Scope type: \"group\" or \"user\".",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scope_id": schema.StringAttribute{
				Description: "Scope identifier (group or user ID).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"recall": schema.BoolAttribute{
				Description: "Recall permission override.",
				Optional:    true,
			},
			"retain": schema.BoolAttribute{
				Description: "Retain permission override.",
				Optional:    true,
			},
			"retain_tags": schema.ListAttribute{
				Description: "Tags to inject on retain.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"retain_roles": schema.ListAttribute{
				Description: "Roles to retain.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"retain_every_n_turns": schema.Int64Attribute{
				Description: "Retain frequency.",
				Optional:    true,
			},
			"recall_budget": schema.StringAttribute{
				Description: "Budget level (low/mid/high).",
				Optional:    true,
			},
			"recall_max_tokens": schema.Int64Attribute{
				Description: "Max recall tokens.",
				Optional:    true,
			},
			"recall_tag_groups": schema.StringAttribute{
				Description: "Tag group filter (JSON-encoded).",
				Optional:    true,
			},
			"llm_model": schema.StringAttribute{
				Description: "LLM model override.",
				Optional:    true,
			},
			"llm_provider": schema.StringAttribute{
				Description: "LLM provider override.",
				Optional:    true,
			},
			"exclude_providers": schema.ListAttribute{
				Description: "Excluded memory providers.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"retain_strategy": schema.StringAttribute{
				Description: "Retain strategy override.",
				Optional:    true,
			},
		},
	}
}

func (r *bankPermissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *bankPermissionResource) buildRequest(ctx context.Context, plan *bankPermissionResourceModel, diags *diag.Diagnostics) *hindclaw.BankPermissionRequest {
	req := hindclaw.NewBankPermissionRequest()

	if !plan.Recall.IsNull() {
		req.SetRecall(plan.Recall.ValueBool())
	} else {
		req.SetRecallNil()
	}
	if !plan.Retain.IsNull() {
		req.SetRetain(plan.Retain.ValueBool())
	} else {
		req.SetRetainNil()
	}
	if !plan.RecallBudget.IsNull() {
		req.SetRecallBudget(plan.RecallBudget.ValueString())
	} else {
		req.SetRecallBudgetNil()
	}
	if !plan.RetainStrategy.IsNull() {
		req.SetRetainStrategy(plan.RetainStrategy.ValueString())
	} else {
		req.SetRetainStrategyNil()
	}
	if !plan.LlmModel.IsNull() {
		req.SetLlmModel(plan.LlmModel.ValueString())
	} else {
		req.SetLlmModelNil()
	}
	if !plan.LlmProvider.IsNull() {
		req.SetLlmProvider(plan.LlmProvider.ValueString())
	} else {
		req.SetLlmProviderNil()
	}
	if !plan.RetainEveryNTurns.IsNull() {
		req.SetRetainEveryNTurns(int32(plan.RetainEveryNTurns.ValueInt64()))
	} else {
		req.SetRetainEveryNTurnsNil()
	}
	if !plan.RecallMaxTokens.IsNull() {
		req.SetRecallMaxTokens(int32(plan.RecallMaxTokens.ValueInt64()))
	} else {
		req.SetRecallMaxTokensNil()
	}
	// List fields: only set when non-null. Nil omits from JSON, which means
	// "no override" for this permission scope. This differs from group_resource
	// Update where empty slice is needed to explicitly clear an existing value.
	if !plan.RetainTags.IsNull() {
		var tags []string
		d := plan.RetainTags.ElementsAs(ctx, &tags, false)
		diags.Append(d...)
		req.SetRetainTags(tags)
	}
	if !plan.RetainRoles.IsNull() {
		var roles []string
		d := plan.RetainRoles.ElementsAs(ctx, &roles, false)
		diags.Append(d...)
		req.SetRetainRoles(roles)
	}
	if !plan.ExcludeProviders.IsNull() {
		var providers []string
		d := plan.ExcludeProviders.ElementsAs(ctx, &providers, false)
		diags.Append(d...)
		req.SetExcludeProviders(providers)
	}
	if !plan.RecallTagGroups.IsNull() {
		var tagGroups []map[string]interface{}
		if err := json.Unmarshal([]byte(plan.RecallTagGroups.ValueString()), &tagGroups); err != nil {
			diags.AddError("Invalid recall_tag_groups JSON", err.Error())
			return nil
		}
		req.SetRecallTagGroups(tagGroups)
	}

	return req
}

func (r *bankPermissionResource) upsert(ctx context.Context, plan *bankPermissionResourceModel, permReq hindclaw.BankPermissionRequest) error {
	bankId := plan.BankID.ValueString()
	scopeId := plan.ScopeID.ValueString()

	switch plan.ScopeType.ValueString() {
	case "group":
		_, _, err := r.client.DefaultAPI.UpsertGroupPermission(ctx, bankId, scopeId).BankPermissionRequest(permReq).Execute()
		return err
	case "user":
		_, _, err := r.client.DefaultAPI.UpsertUserPermission(ctx, bankId, scopeId).BankPermissionRequest(permReq).Execute()
		return err
	default:
		return fmt.Errorf("scope_type must be \"group\" or \"user\", got %q", plan.ScopeType.ValueString())
	}
}

func (r *bankPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bankPermissionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	permReq := r.buildRequest(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.upsert(ctx, &plan, *permReq); err != nil {
		resp.Diagnostics.AddError("Error creating bank permission", err.Error())
		return
	}

	// Read back
	if notFound := r.readPermissionIntoState(ctx, &plan, &resp.Diagnostics); notFound {
		resp.Diagnostics.AddError("Error reading bank permission after create", "Permission not found immediately after creation")
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created bank permission", map[string]any{
		"bank_id":    plan.BankID.ValueString(),
		"scope_type": plan.ScopeType.ValueString(),
		"scope_id":   plan.ScopeID.ValueString(),
	})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *bankPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bankPermissionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if notFound := r.readPermissionIntoState(ctx, &state, &resp.Diagnostics); notFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *bankPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bankPermissionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	permReq := r.buildRequest(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.upsert(ctx, &plan, *permReq); err != nil {
		resp.Diagnostics.AddError("Error updating bank permission", err.Error())
		return
	}

	if notFound := r.readPermissionIntoState(ctx, &plan, &resp.Diagnostics); notFound {
		resp.Diagnostics.AddError("Error reading bank permission after update", "Permission not found immediately after update")
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *bankPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bankPermissionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.DefaultAPI.DeleteBankPermission(ctx,
		state.BankID.ValueString(),
		state.ScopeType.ValueString(),
		state.ScopeID.ValueString(),
	).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting bank permission", err.Error())
		return
	}
}

func (r *bankPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: {bank_id}/{scope_type}/{scope_id}")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bank_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope_type"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope_id"), parts[2])...)
}

func (r *bankPermissionResource) readPermissionIntoState(ctx context.Context, state *bankPermissionResourceModel, diags *diag.Diagnostics) (notFound bool) {
	perm, httpResp, err := r.client.DefaultAPI.GetBankPermission(ctx,
		state.BankID.ValueString(),
		state.ScopeType.ValueString(),
		state.ScopeID.ValueString(),
	).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return true
		}
		diags.AddError("Error reading bank permission", err.Error())
		return false
	}

	state.BankID = types.StringValue(perm.BankId)
	state.ScopeType = types.StringValue(perm.ScopeType)
	state.ScopeID = types.StringValue(perm.ScopeId)

	state.Recall = nullableBoolToTF(perm.Recall)
	state.Retain = nullableBoolToTF(perm.Retain)
	state.RecallBudget = nullableStringToTF(perm.RecallBudget)
	state.RetainStrategy = nullableStringToTF(perm.RetainStrategy)
	state.LlmModel = nullableStringToTF(perm.LlmModel)
	state.LlmProvider = nullableStringToTF(perm.LlmProvider)
	state.RetainEveryNTurns = nullableInt32ToTF(perm.RetainEveryNTurns)
	state.RecallMaxTokens = nullableInt32ToTF(perm.RecallMaxTokens)

	if perm.RetainTags != nil {
		listVal, d := types.ListValueFrom(ctx, types.StringType, perm.RetainTags)
		diags.Append(d...)
		state.RetainTags = listVal
	} else {
		state.RetainTags = types.ListNull(types.StringType)
	}
	if perm.RetainRoles != nil {
		listVal, d := types.ListValueFrom(ctx, types.StringType, perm.RetainRoles)
		diags.Append(d...)
		state.RetainRoles = listVal
	} else {
		state.RetainRoles = types.ListNull(types.StringType)
	}
	if perm.ExcludeProviders != nil {
		listVal, d := types.ListValueFrom(ctx, types.StringType, perm.ExcludeProviders)
		diags.Append(d...)
		state.ExcludeProviders = listVal
	} else {
		state.ExcludeProviders = types.ListNull(types.StringType)
	}
	if perm.RecallTagGroups != nil {
		jsonBytes, err := json.Marshal(perm.RecallTagGroups)
		if err != nil {
			diags.AddError("Error marshaling recall_tag_groups", err.Error())
			return false
		}
		state.RecallTagGroups = types.StringValue(string(jsonBytes))
	} else {
		state.RecallTagGroups = types.StringNull()
	}

	return false
}
