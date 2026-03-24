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
	_ resource.Resource                = &bankPolicyResource{}
	_ resource.ResourceWithConfigure   = &bankPolicyResource{}
	_ resource.ResourceWithImportState = &bankPolicyResource{}
)

func NewBankPolicyResource() resource.Resource {
	return &bankPolicyResource{}
}

type bankPolicyResourceModel struct {
	BankID   types.String `tfsdk:"bank_id"`
	Document types.String `tfsdk:"document"`
}

type bankPolicyResource struct {
	client *hindclaw.APIClient
}

func (r *bankPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bank_policy"
}

func (r *bankPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a bank policy document. One policy per bank.",
		Attributes: map[string]schema.Attribute{
			"bank_id": schema.StringAttribute{
				Description: "Bank identifier. Immutable after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"document": schema.StringAttribute{
				Description: "Policy document as a JSON string.",
				Required:    true,
			},
		},
	}
}

func (r *bankPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *bankPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bankPolicyResourceModel
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

	upsertReq := hindclaw.NewUpsertBankPolicyRequest(docMap)

	policy, _, err := r.client.DefaultAPI.UpsertBankPolicy(ctx, plan.BankID.ValueString()).
		UpsertBankPolicyRequest(*upsertReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating bank policy", err.Error())
		return
	}

	plan.BankID = types.StringValue(policy.BankId)

	docBytes, err := json.Marshal(policy.Document)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling bank policy document", err.Error())
		return
	}
	plan.Document = types.StringValue(string(docBytes))

	tflog.Trace(ctx, "created bank policy", map[string]any{"bank_id": policy.BankId})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *bankPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bankPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, httpResp, err := r.client.DefaultAPI.GetBankPolicy(ctx, state.BankID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading bank policy", err.Error())
		return
	}

	state.BankID = types.StringValue(policy.BankId)

	docBytes, err := json.Marshal(policy.Document)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling bank policy document", err.Error())
		return
	}
	state.Document = types.StringValue(string(docBytes))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *bankPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bankPolicyResourceModel
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

	upsertReq := hindclaw.NewUpsertBankPolicyRequest(docMap)

	policy, _, err := r.client.DefaultAPI.UpsertBankPolicy(ctx, plan.BankID.ValueString()).
		UpsertBankPolicyRequest(*upsertReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating bank policy", err.Error())
		return
	}

	plan.BankID = types.StringValue(policy.BankId)

	docBytes, err := json.Marshal(policy.Document)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling bank policy document", err.Error())
		return
	}
	plan.Document = types.StringValue(string(docBytes))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *bankPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bankPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.DefaultAPI.DeleteBankPolicy(ctx, state.BankID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting bank policy", err.Error())
		return
	}
}

func (r *bankPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bank_id"), req, resp)
}
