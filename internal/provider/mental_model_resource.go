package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	hindsight "github.com/vectorize-io/hindsight/hindsight-clients/go"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &mentalModelResource{}
	_ resource.ResourceWithConfigure   = &mentalModelResource{}
	_ resource.ResourceWithImportState = &mentalModelResource{}
)

func NewMentalModelResource() resource.Resource {
	return &mentalModelResource{}
}

type mentalModelResourceModel struct {
	BankID      types.String `tfsdk:"bank_id"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	SourceQuery types.String `tfsdk:"source_query"`
	Tags        types.List   `tfsdk:"tags"`
	MaxTokens   types.Int64  `tfsdk:"max_tokens"`
	Trigger     types.Object `tfsdk:"trigger"`
}

type mentalModelTriggerModel struct {
	RefreshAfterConsolidation types.Bool `tfsdk:"refresh_after_consolidation"`
	FactTypes                 types.List `tfsdk:"fact_types"`
	ExcludeMentalModels       types.Bool `tfsdk:"exclude_mental_models"`
	ExcludeMentalModelIds     types.List `tfsdk:"exclude_mental_model_ids"`
}

// triggerObjectAttrTypes is the type map for the trigger nested object.
// Shared between ObjectValue (read) and ObjectNull (defensive nil branch).
var triggerObjectAttrTypes = map[string]attr.Type{
	"refresh_after_consolidation": types.BoolType,
	"exclude_mental_models":       types.BoolType,
	"fact_types":                  types.ListType{ElemType: types.StringType},
	"exclude_mental_model_ids":    types.ListType{ElemType: types.StringType},
}

type mentalModelResource struct {
	client *hindsight.APIClient
}

func (r *mentalModelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mental_model"
}

func (r *mentalModelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hindsight mental model.",
		Attributes: map[string]schema.Attribute{
			"bank_id": schema.StringAttribute{
				Description: "Bank identifier.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "Mental model identifier (computed by server).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Mental model name.",
				Required:    true,
			},
			"source_query": schema.StringAttribute{
				Description: "Query used to build the model.",
				Required:    true,
			},
			"tags": schema.ListAttribute{
				Description: "Tags for the mental model.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"max_tokens": schema.Int64Attribute{
				Description: "Max tokens for the model content (default 2048).",
				Optional:    true,
				Computed:    true,
			},
			"trigger": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Auto-refresh trigger configuration.",
				Attributes: map[string]schema.Attribute{
					"refresh_after_consolidation": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Auto-refresh this mental model after observation consolidation.",
						Default:     booldefault.StaticBool(false),
					},
					"fact_types": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Restrict which fact types the refresh retrieves. Subset of [\"world\", \"experience\", \"observation\"].",
					},
					"exclude_mental_models": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Skip mental model search during refresh.",
						Default:     booldefault.StaticBool(false),
					},
					"exclude_mental_model_ids": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Exclude specific mental models by ID during refresh.",
					},
				},
			},
		},
	}
}

func (r *mentalModelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *mentalModelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mentalModelResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := hindsight.NewCreateMentalModelRequest(plan.Name.ValueString(), plan.SourceQuery.ValueString())
	if !plan.Tags.IsNull() {
		var tags []string
		diags = plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		createReq.SetTags(tags)
	}
	if !plan.MaxTokens.IsNull() {
		createReq.SetMaxTokens(int32(plan.MaxTokens.ValueInt64()))
	}
	if !plan.Trigger.IsNull() && !plan.Trigger.IsUnknown() {
		var trigger mentalModelTriggerModel
		diags = plan.Trigger.As(ctx, &trigger, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		apiTrigger, triggerDiags := triggerModelToAPI(ctx, trigger)
		resp.Diagnostics.Append(triggerDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.SetTrigger(*apiTrigger)
	}

	createResp, _, err := r.client.MentalModelsAPI.CreateMentalModel(ctx, plan.BankID.ValueString()).CreateMentalModelRequest(*createReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating mental model", err.Error())
		return
	}

	modelId := ""
	if id, ok := requiredNullableString(createResp.MentalModelId); ok {
		modelId = id
	}
	if modelId == "" {
		resp.Diagnostics.AddError("Error creating mental model", "Server returned empty mental_model_id")
		return
	}

	// Read back full state
	model, _, err := r.client.MentalModelsAPI.GetMentalModel(ctx, plan.BankID.ValueString(), modelId).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error reading mental model after create", err.Error())
		return
	}

	r.readResponseIntoState(ctx, model, &plan, &resp.Diagnostics)

	tflog.Trace(ctx, "created mental model", map[string]any{"id": modelId})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *mentalModelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mentalModelResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	model, httpResp, err := r.client.MentalModelsAPI.GetMentalModel(ctx, state.BankID.ValueString(), state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading mental model", err.Error())
		return
	}

	r.readResponseIntoState(ctx, model, &state, &resp.Diagnostics)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *mentalModelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mentalModelResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := hindsight.NewUpdateMentalModelRequest()
	updateReq.SetName(plan.Name.ValueString())
	updateReq.SetSourceQuery(plan.SourceQuery.ValueString())
	if !plan.Tags.IsNull() {
		var tags []string
		diags = plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		updateReq.SetTags(tags)
	} else {
		updateReq.SetTags([]string{})
	}
	if !plan.MaxTokens.IsNull() {
		updateReq.SetMaxTokens(int32(plan.MaxTokens.ValueInt64()))
	} else {
		updateReq.SetMaxTokensNil()
	}
	if !plan.Trigger.IsNull() && !plan.Trigger.IsUnknown() {
		var trigger mentalModelTriggerModel
		diags = plan.Trigger.As(ctx, &trigger, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		apiTrigger, triggerDiags := triggerModelToAPI(ctx, trigger)
		resp.Diagnostics.Append(triggerDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.SetTrigger(*apiTrigger)
	} else if plan.Trigger.IsNull() {
		// Reset to server default. SetTriggerNil() is NOT usable —
		// upstream skips the update when trigger is None.
		updateReq.SetTrigger(*hindsight.NewMentalModelTrigger())
	}

	model, _, err := r.client.MentalModelsAPI.UpdateMentalModel(ctx, plan.BankID.ValueString(), plan.ID.ValueString()).UpdateMentalModelRequest(*updateReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating mental model", err.Error())
		return
	}

	r.readResponseIntoState(ctx, model, &plan, &resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *mentalModelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mentalModelResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, httpResp, err := r.client.MentalModelsAPI.DeleteMentalModel(ctx, state.BankID.ValueString(), state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting mental model", err.Error())
		return
	}
}

func (r *mentalModelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: {bank_id}/{id}")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bank_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// triggerModelToAPI converts a Terraform trigger model to the Hindsight API trigger.
func triggerModelToAPI(ctx context.Context, trigger mentalModelTriggerModel) (*hindsight.MentalModelTrigger, diag.Diagnostics) {
	var diags diag.Diagnostics
	apiTrigger := hindsight.NewMentalModelTrigger()
	if !trigger.RefreshAfterConsolidation.IsNull() {
		apiTrigger.SetRefreshAfterConsolidation(trigger.RefreshAfterConsolidation.ValueBool())
	}
	if !trigger.FactTypes.IsNull() {
		var factTypes []string
		diags.Append(trigger.FactTypes.ElementsAs(ctx, &factTypes, false)...)
		if diags.HasError() {
			return apiTrigger, diags
		}
		apiTrigger.SetFactTypes(factTypes)
	}
	if !trigger.ExcludeMentalModels.IsNull() {
		apiTrigger.SetExcludeMentalModels(trigger.ExcludeMentalModels.ValueBool())
	}
	if !trigger.ExcludeMentalModelIds.IsNull() {
		var ids []string
		diags.Append(trigger.ExcludeMentalModelIds.ElementsAs(ctx, &ids, false)...)
		if diags.HasError() {
			return apiTrigger, diags
		}
		apiTrigger.SetExcludeMentalModelIds(ids)
	}
	return apiTrigger, diags
}

func (r *mentalModelResource) readResponseIntoState(ctx context.Context, model *hindsight.MentalModelResponse, state *mentalModelResourceModel, diags *diag.Diagnostics) {
	state.BankID = types.StringValue(model.BankId)
	state.ID = types.StringValue(model.Id)
	state.Name = types.StringValue(model.Name)
	state.SourceQuery = types.StringValue(model.SourceQuery)

	if model.Tags != nil {
		listVal, d := stringSliceToTFListPreserveNullOnEmpty(ctx, state.Tags, model.Tags)
		diags.Append(d...)
		state.Tags = listVal
	} else {
		state.Tags = types.ListNull(types.StringType)
	}
	if model.MaxTokens != nil {
		state.MaxTokens = types.Int64Value(int64(*model.MaxTokens))
	} else {
		state.MaxTokens = types.Int64Null()
	}
	if model.Trigger != nil {
		trigger := model.Trigger

		// Extract prior trigger lists for null preservation
		var priorFactTypes, priorExcludeIds types.List
		priorFactTypes = types.ListNull(types.StringType)
		priorExcludeIds = types.ListNull(types.StringType)
		if !state.Trigger.IsNull() && !state.Trigger.IsUnknown() {
			var prior mentalModelTriggerModel
			diags.Append(state.Trigger.As(ctx, &prior, basetypes.ObjectAsOptions{})...)
			priorFactTypes = prior.FactTypes
			priorExcludeIds = prior.ExcludeMentalModelIds
		}

		factTypesList, d := stringSliceToTFListPreserveNullOnEmpty(ctx, priorFactTypes, trigger.GetFactTypes())
		diags.Append(d...)
		excludeIdsList, d := stringSliceToTFListPreserveNullOnEmpty(ctx, priorExcludeIds, trigger.GetExcludeMentalModelIds())
		diags.Append(d...)

		triggerAttrs := map[string]attr.Value{
			"refresh_after_consolidation": types.BoolValue(trigger.GetRefreshAfterConsolidation()),
			"exclude_mental_models":       types.BoolValue(trigger.GetExcludeMentalModels()),
			"fact_types":                  factTypesList,
			"exclude_mental_model_ids":    excludeIdsList,
		}
		triggerObj, d := types.ObjectValue(triggerObjectAttrTypes, triggerAttrs)
		diags.Append(d...)
		state.Trigger = triggerObj
	} else {
		// Defensive: server always returns a trigger, but handle nil gracefully.
		state.Trigger = types.ObjectNull(triggerObjectAttrTypes)
	}
}
