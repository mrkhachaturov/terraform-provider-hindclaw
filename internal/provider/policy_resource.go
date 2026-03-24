package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
	_ resource.Resource                = &policyResource{}
	_ resource.ResourceWithConfigure   = &policyResource{}
	_ resource.ResourceWithImportState = &policyResource{}
)

func NewPolicyResource() resource.Resource {
	return &policyResource{}
}

type policyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	DisplayName types.String `tfsdk:"display_name"`
	Document    types.String `tfsdk:"document"`
}

type policyResource struct {
	client *hindclaw.APIClient
}

func (r *policyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *policyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hindclaw access policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Policy identifier. Immutable after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "Display name.",
				Required:    true,
			},
			"document": schema.StringAttribute{
				Description: "Policy document as a JSON string.",
				Required:    true,
			},
		},
	}
}

func (r *policyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan policyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var docMap map[string]interface{}
	if err := json.Unmarshal([]byte(plan.Document.ValueString()), &docMap); err != nil {
		resp.Diagnostics.AddError("Invalid document JSON", err.Error())
		return
	}

	createReq := hindclaw.NewCreatePolicyRequest(
		plan.ID.ValueString(),
		plan.DisplayName.ValueString(),
		docMap,
	)

	policy, _, err := r.client.DefaultAPI.CreatePolicy(ctx).CreatePolicyRequest(*createReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating policy", err.Error())
		return
	}

	docBytes, err := json.Marshal(policy.Document)
	if err != nil {
		resp.Diagnostics.AddError("Error serialising document", err.Error())
		return
	}

	plan.ID = types.StringValue(policy.Id)
	plan.DisplayName = types.StringValue(policy.DisplayName)
	plan.Document = types.StringValue(string(docBytes))

	tflog.Trace(ctx, "created policy", map[string]any{"id": policy.Id})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *policyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state policyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, httpResp, err := r.client.DefaultAPI.GetPolicy(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading policy", err.Error())
		return
	}

	docBytes, err := json.Marshal(policy.Document)
	if err != nil {
		resp.Diagnostics.AddError("Error serialising document", err.Error())
		return
	}

	state.ID = types.StringValue(policy.Id)
	state.DisplayName = types.StringValue(policy.DisplayName)
	state.Document = types.StringValue(string(docBytes))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *policyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan policyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var docMap map[string]interface{}
	if err := json.Unmarshal([]byte(plan.Document.ValueString()), &docMap); err != nil {
		resp.Diagnostics.AddError("Invalid document JSON", err.Error())
		return
	}

	updateReq := hindclaw.NewUpdatePolicyRequest()
	updateReq.SetDisplayName(plan.DisplayName.ValueString())
	updateReq.SetDocument(docMap)

	policy, _, err := r.client.DefaultAPI.UpdatePolicy(ctx, plan.ID.ValueString()).UpdatePolicyRequest(*updateReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating policy", err.Error())
		return
	}

	docBytes, err := json.Marshal(policy.Document)
	if err != nil {
		resp.Diagnostics.AddError("Error serialising document", err.Error())
		return
	}

	plan.ID = types.StringValue(policy.Id)
	plan.DisplayName = types.StringValue(policy.DisplayName)
	plan.Document = types.StringValue(string(docBytes))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *policyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state policyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.DefaultAPI.DeletePolicy(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting policy", err.Error())
		return
	}
}

func (r *policyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
