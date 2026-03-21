package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
	_ resource.Resource                = &groupResource{}
	_ resource.ResourceWithConfigure   = &groupResource{}
	_ resource.ResourceWithImportState = &groupResource{}
)

func NewGroupResource() resource.Resource {
	return &groupResource{}
}

type groupResourceModel struct {
	ID                types.String `tfsdk:"id"`
	DisplayName       types.String `tfsdk:"display_name"`
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

type groupResource struct {
	client *hindclaw.APIClient
}

func (r *groupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *groupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hindclaw group with permission defaults.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Group identifier. Immutable after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "Display name.",
				Required:    true,
			},
			"recall": schema.BoolAttribute{
				Description: "Global recall default.",
				Optional:    true,
			},
			"retain": schema.BoolAttribute{
				Description: "Global retain default.",
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
				Description: "Default retain strategy.",
				Optional:    true,
			},
		},
	}
}

func (r *groupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := hindclaw.NewCreateGroupRequest(plan.ID.ValueString(), plan.DisplayName.ValueString())

	if !plan.Recall.IsNull() {
		createReq.SetRecall(plan.Recall.ValueBool())
	}
	if !plan.Retain.IsNull() {
		createReq.SetRetain(plan.Retain.ValueBool())
	}
	if !plan.RecallBudget.IsNull() {
		createReq.SetRecallBudget(plan.RecallBudget.ValueString())
	}
	if !plan.RetainStrategy.IsNull() {
		createReq.SetRetainStrategy(plan.RetainStrategy.ValueString())
	}
	if !plan.LlmModel.IsNull() {
		createReq.SetLlmModel(plan.LlmModel.ValueString())
	}
	if !plan.LlmProvider.IsNull() {
		createReq.SetLlmProvider(plan.LlmProvider.ValueString())
	}
	// List fields: convert Terraform types.List -> Go []string
	if !plan.RetainTags.IsNull() {
		var tags []string
		diags = plan.RetainTags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			createReq.SetRetainTags(tags)
		}
	}
	if !plan.RetainRoles.IsNull() {
		var roles []string
		diags = plan.RetainRoles.ElementsAs(ctx, &roles, false)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			createReq.SetRetainRoles(roles)
		}
	}
	if !plan.ExcludeProviders.IsNull() {
		var providers []string
		diags = plan.ExcludeProviders.ElementsAs(ctx, &providers, false)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			createReq.SetExcludeProviders(providers)
		}
	}
	// recall_tag_groups: JSON string -> []map[string]interface{}
	if !plan.RecallTagGroups.IsNull() {
		var tagGroups []map[string]interface{}
		if err := json.Unmarshal([]byte(plan.RecallTagGroups.ValueString()), &tagGroups); err != nil {
			resp.Diagnostics.AddError("Invalid recall_tag_groups JSON", err.Error())
			return
		}
		createReq.SetRecallTagGroups(tagGroups)
	}
	if !plan.RetainEveryNTurns.IsNull() {
		createReq.SetRetainEveryNTurns(int32(plan.RetainEveryNTurns.ValueInt64()))
	}
	if !plan.RecallMaxTokens.IsNull() {
		createReq.SetRecallMaxTokens(int32(plan.RecallMaxTokens.ValueInt64()))
	}

	_, _, err := r.client.DefaultAPI.CreateGroup(ctx).CreateGroupRequest(*createReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating group", err.Error())
		return
	}

	// Read back full state from server
	if notFound := r.readGroupIntoState(ctx, plan.ID.ValueString(), &plan, &resp.Diagnostics); notFound {
		resp.Diagnostics.AddError("Error reading group after create", "Group not found immediately after creation")
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created group", map[string]any{"id": plan.ID.ValueString()})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if notFound := r.readGroupIntoState(ctx, state.ID.ValueString(), &state, &resp.Diagnostics); notFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan groupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := hindclaw.NewUpdateGroupRequest()
	updateReq.SetDisplayName(plan.DisplayName.ValueString())

	// For each nullable field: set value if non-null, send nil to clear if null.
	// Without the nil branch, removing a field from Terraform config would never
	// clear it on the server — the API treats omitted fields as "leave unchanged".
	if !plan.Recall.IsNull() {
		updateReq.SetRecall(plan.Recall.ValueBool())
	} else {
		updateReq.SetRecallNil()
	}
	if !plan.Retain.IsNull() {
		updateReq.SetRetain(plan.Retain.ValueBool())
	} else {
		updateReq.SetRetainNil()
	}
	if !plan.RecallBudget.IsNull() {
		updateReq.SetRecallBudget(plan.RecallBudget.ValueString())
	} else {
		updateReq.SetRecallBudgetNil()
	}
	if !plan.RetainStrategy.IsNull() {
		updateReq.SetRetainStrategy(plan.RetainStrategy.ValueString())
	} else {
		updateReq.SetRetainStrategyNil()
	}
	if !plan.LlmModel.IsNull() {
		updateReq.SetLlmModel(plan.LlmModel.ValueString())
	} else {
		updateReq.SetLlmModelNil()
	}
	if !plan.LlmProvider.IsNull() {
		updateReq.SetLlmProvider(plan.LlmProvider.ValueString())
	} else {
		updateReq.SetLlmProviderNil()
	}
	if !plan.RetainEveryNTurns.IsNull() {
		updateReq.SetRetainEveryNTurns(int32(plan.RetainEveryNTurns.ValueInt64()))
	} else {
		updateReq.SetRetainEveryNTurnsNil()
	}
	if !plan.RecallMaxTokens.IsNull() {
		updateReq.SetRecallMaxTokens(int32(plan.RecallMaxTokens.ValueInt64()))
	} else {
		updateReq.SetRecallMaxTokensNil()
	}
	// List fields: set value or send empty slice to clear.
	// The generated client omits nil slices from JSON (nil = "leave unchanged").
	// An empty slice serializes as [] which tells the server to clear the field.
	if !plan.RetainTags.IsNull() {
		var tags []string
		diags = plan.RetainTags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			updateReq.SetRetainTags(tags)
		}
	} else {
		updateReq.SetRetainTags([]string{})
	}
	if !plan.RetainRoles.IsNull() {
		var roles []string
		diags = plan.RetainRoles.ElementsAs(ctx, &roles, false)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			updateReq.SetRetainRoles(roles)
		}
	} else {
		updateReq.SetRetainRoles([]string{})
	}
	if !plan.ExcludeProviders.IsNull() {
		var providers []string
		diags = plan.ExcludeProviders.ElementsAs(ctx, &providers, false)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			updateReq.SetExcludeProviders(providers)
		}
	} else {
		updateReq.SetExcludeProviders([]string{})
	}
	if !plan.RecallTagGroups.IsNull() {
		var tagGroups []map[string]interface{}
		if err := json.Unmarshal([]byte(plan.RecallTagGroups.ValueString()), &tagGroups); err != nil {
			resp.Diagnostics.AddError("Invalid recall_tag_groups JSON", err.Error())
			return
		}
		updateReq.SetRecallTagGroups(tagGroups)
	} else {
		updateReq.SetRecallTagGroups([]map[string]interface{}{})
	}

	_, _, err := r.client.DefaultAPI.UpdateGroup(ctx, plan.ID.ValueString()).UpdateGroupRequest(*updateReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating group", err.Error())
		return
	}

	// Read back full state
	if notFound := r.readGroupIntoState(ctx, plan.ID.ValueString(), &plan, &resp.Diagnostics); notFound {
		resp.Diagnostics.AddError("Error reading group after update", "Group not found immediately after update")
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.DefaultAPI.DeleteGroup(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting group", err.Error())
		return
	}
}

func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// readGroupIntoState fetches the group from the API and populates the model.
// Returns true if the group was not found (404), so callers can remove from state.
func (r *groupResource) readGroupIntoState(ctx context.Context, groupID string, state *groupResourceModel, diags *diag.Diagnostics) (notFound bool) {
	group, httpResp, err := r.client.DefaultAPI.GetGroup(ctx, groupID).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return true
		}
		diags.AddError("Error reading group", err.Error())
		return false
	}

	state.ID = types.StringValue(group.Id)
	state.DisplayName = types.StringValue(group.DisplayName)

	if group.Recall.IsSet() {
		state.Recall = types.BoolValue(*group.Recall.Get())
	} else {
		state.Recall = types.BoolNull()
	}
	if group.Retain.IsSet() {
		state.Retain = types.BoolValue(*group.Retain.Get())
	} else {
		state.Retain = types.BoolNull()
	}
	if group.RecallBudget.IsSet() {
		state.RecallBudget = types.StringValue(*group.RecallBudget.Get())
	} else {
		state.RecallBudget = types.StringNull()
	}
	if group.RetainStrategy.IsSet() {
		state.RetainStrategy = types.StringValue(*group.RetainStrategy.Get())
	} else {
		state.RetainStrategy = types.StringNull()
	}
	if group.LlmModel.IsSet() {
		state.LlmModel = types.StringValue(*group.LlmModel.Get())
	} else {
		state.LlmModel = types.StringNull()
	}
	if group.LlmProvider.IsSet() {
		state.LlmProvider = types.StringValue(*group.LlmProvider.Get())
	} else {
		state.LlmProvider = types.StringNull()
	}
	if group.RetainEveryNTurns.IsSet() {
		state.RetainEveryNTurns = types.Int64Value(int64(*group.RetainEveryNTurns.Get()))
	} else {
		state.RetainEveryNTurns = types.Int64Null()
	}
	if group.RecallMaxTokens.IsSet() {
		state.RecallMaxTokens = types.Int64Value(int64(*group.RecallMaxTokens.Get()))
	} else {
		state.RecallMaxTokens = types.Int64Null()
	}

	// List fields: convert Go slices to Terraform types.List.
	// Use group.RetainTags != nil (not len > 0) to distinguish "server returned
	// empty list" from "server returned null/omitted". This prevents perpetual
	// diffs when Terraform config has [] — refresh must return [] not null.
	if group.RetainTags != nil {
		listVal, d := types.ListValueFrom(ctx, types.StringType, group.RetainTags)
		diags.Append(d...)
		state.RetainTags = listVal
	} else {
		state.RetainTags = types.ListNull(types.StringType)
	}
	if group.RetainRoles != nil {
		listVal, d := types.ListValueFrom(ctx, types.StringType, group.RetainRoles)
		diags.Append(d...)
		state.RetainRoles = listVal
	} else {
		state.RetainRoles = types.ListNull(types.StringType)
	}
	if group.ExcludeProviders != nil {
		listVal, d := types.ListValueFrom(ctx, types.StringType, group.ExcludeProviders)
		diags.Append(d...)
		state.ExcludeProviders = listVal
	} else {
		state.ExcludeProviders = types.ListNull(types.StringType)
	}
	// recall_tag_groups: []map[string]interface{} -> JSON string.
	// Same nil-vs-empty distinction: nil -> StringNull, [] -> "[]".
	if group.RecallTagGroups != nil {
		jsonBytes, err := json.Marshal(group.RecallTagGroups)
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
