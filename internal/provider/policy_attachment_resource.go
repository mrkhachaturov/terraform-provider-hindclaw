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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &policyAttachmentResource{}
	_ resource.ResourceWithConfigure   = &policyAttachmentResource{}
	_ resource.ResourceWithImportState = &policyAttachmentResource{}
)

func NewPolicyAttachmentResource() resource.Resource {
	return &policyAttachmentResource{}
}

type policyAttachmentResourceModel struct {
	PolicyID      types.String `tfsdk:"policy_id"`
	PrincipalType types.String `tfsdk:"principal_type"`
	PrincipalID   types.String `tfsdk:"principal_id"`
	Priority      types.Int64  `tfsdk:"priority"`
}

type policyAttachmentResource struct {
	client *hindclaw.APIClient
}

func (r *policyAttachmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_attachment"
}

func (r *policyAttachmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches a policy to a user or group principal.",
		Attributes: map[string]schema.Attribute{
			"policy_id": schema.StringAttribute{
				Description: "Policy identifier.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"principal_type": schema.StringAttribute{
				Description: "Principal type: \"user\" or \"group\".",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"principal_id": schema.StringAttribute{
				Description: "Principal identifier (user or group ID).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"priority": schema.Int64Attribute{
				Description: "Attachment priority. Higher values take precedence.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
		},
	}
}

func (r *policyAttachmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *policyAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan policyAttachmentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := hindclaw.NewCreatePolicyAttachmentRequest(
		plan.PolicyID.ValueString(),
		plan.PrincipalType.ValueString(),
		plan.PrincipalID.ValueString(),
	)
	createReq.SetPriority(int32(plan.Priority.ValueInt64()))

	attachment, _, err := r.client.DefaultAPI.UpsertPolicyAttachment(ctx).
		CreatePolicyAttachmentRequest(*createReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating policy attachment", err.Error())
		return
	}

	plan.PolicyID = types.StringValue(attachment.PolicyId)
	plan.PrincipalType = types.StringValue(attachment.PrincipalType)
	plan.PrincipalID = types.StringValue(attachment.PrincipalId)
	plan.Priority = types.Int64Value(int64(attachment.Priority))

	tflog.Trace(ctx, "created policy attachment", map[string]any{
		"policy_id":      attachment.PolicyId,
		"principal_type": attachment.PrincipalType,
		"principal_id":   attachment.PrincipalId,
	})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *policyAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state policyAttachmentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	attachments, httpResp, err := r.client.DefaultAPI.ListPolicyAttachments(ctx).
		PolicyId(state.PolicyID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading policy attachments", err.Error())
		return
	}

	// Find the matching attachment by principal_type + principal_id.
	var found *hindclaw.PolicyAttachmentResponse
	for i := range attachments {
		a := &attachments[i]
		if a.PrincipalType == state.PrincipalType.ValueString() && a.PrincipalId == state.PrincipalID.ValueString() {
			found = a
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.PolicyID = types.StringValue(found.PolicyId)
	state.PrincipalType = types.StringValue(found.PrincipalType)
	state.PrincipalID = types.StringValue(found.PrincipalId)
	state.Priority = types.Int64Value(int64(found.Priority))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *policyAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan policyAttachmentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	upsertReq := hindclaw.NewCreatePolicyAttachmentRequest(
		plan.PolicyID.ValueString(),
		plan.PrincipalType.ValueString(),
		plan.PrincipalID.ValueString(),
	)
	upsertReq.SetPriority(int32(plan.Priority.ValueInt64()))

	attachment, _, err := r.client.DefaultAPI.UpsertPolicyAttachment(ctx).
		CreatePolicyAttachmentRequest(*upsertReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating policy attachment", err.Error())
		return
	}

	plan.PolicyID = types.StringValue(attachment.PolicyId)
	plan.PrincipalType = types.StringValue(attachment.PrincipalType)
	plan.PrincipalID = types.StringValue(attachment.PrincipalId)
	plan.Priority = types.Int64Value(int64(attachment.Priority))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *policyAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state policyAttachmentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.DefaultAPI.DeletePolicyAttachment(ctx,
		state.PolicyID.ValueString(),
		state.PrincipalType.ValueString(),
		state.PrincipalID.ValueString(),
	).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting policy attachment", err.Error())
		return
	}
}

func (r *policyAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: {policy_id}/{principal_type}/{principal_id}")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_type"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_id"), parts[2])...)
}
