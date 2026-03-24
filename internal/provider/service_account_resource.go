package provider

import (
	"context"
	"fmt"
	"net/http"

	hindclaw "github.com/mrkhachaturov/hindclaw/hindclaw-clients/go"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &serviceAccountResource{}
	_ resource.ResourceWithConfigure   = &serviceAccountResource{}
	_ resource.ResourceWithImportState = &serviceAccountResource{}
)

func NewServiceAccountResource() resource.Resource {
	return &serviceAccountResource{}
}

type serviceAccountResourceModel struct {
	ID              types.String `tfsdk:"id"`
	OwnerUserID     types.String `tfsdk:"owner_user_id"`
	DisplayName     types.String `tfsdk:"display_name"`
	ScopingPolicyID types.String `tfsdk:"scoping_policy_id"`
	IsActive        types.Bool   `tfsdk:"is_active"`
}

type serviceAccountResource struct {
	client *hindclaw.APIClient
}

func (r *serviceAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (r *serviceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hindclaw service account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Service account identifier. Immutable after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner_user_id": schema.StringAttribute{
				Description: "Owner user identifier. Immutable after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "Display name.",
				Required:    true,
			},
			"scoping_policy_id": schema.StringAttribute{
				Description: "Scoping policy identifier. Null means full parent inheritance.",
				Optional:    true,
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether the service account is active (read from server).",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *serviceAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serviceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceAccountResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := hindclaw.NewCreateServiceAccountRequest(plan.ID.ValueString(), plan.OwnerUserID.ValueString(), plan.DisplayName.ValueString())
	if !plan.ScopingPolicyID.IsNull() {
		createReq.SetScopingPolicyId(plan.ScopingPolicyID.ValueString())
	}

	sa, _, err := r.client.DefaultAPI.CreateServiceAccount(ctx).CreateServiceAccountRequest(*createReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating service account", err.Error())
		return
	}

	plan.ID = types.StringValue(sa.Id)
	plan.OwnerUserID = types.StringValue(sa.OwnerUserId)
	plan.DisplayName = types.StringValue(sa.DisplayName)
	plan.ScopingPolicyID = nullableStringToTF(sa.ScopingPolicyId)
	plan.IsActive = types.BoolValue(sa.IsActive)

	tflog.Trace(ctx, "created service account", map[string]any{"id": sa.Id})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *serviceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceAccountResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sa, httpResp, err := r.client.DefaultAPI.GetServiceAccount(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading service account", err.Error())
		return
	}

	state.ID = types.StringValue(sa.Id)
	state.OwnerUserID = types.StringValue(sa.OwnerUserId)
	state.DisplayName = types.StringValue(sa.DisplayName)
	state.ScopingPolicyID = nullableStringToTF(sa.ScopingPolicyId)
	state.IsActive = types.BoolValue(sa.IsActive)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *serviceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceAccountResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := hindclaw.NewUpdateServiceAccountRequest()
	updateReq.SetDisplayName(plan.DisplayName.ValueString())
	if !plan.ScopingPolicyID.IsNull() {
		updateReq.SetScopingPolicyId(plan.ScopingPolicyID.ValueString())
	} else {
		updateReq.SetScopingPolicyIdNil()
	}

	sa, _, err := r.client.DefaultAPI.UpdateServiceAccount(ctx, plan.ID.ValueString()).UpdateServiceAccountRequest(*updateReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating service account", err.Error())
		return
	}

	plan.ID = types.StringValue(sa.Id)
	plan.OwnerUserID = types.StringValue(sa.OwnerUserId)
	plan.DisplayName = types.StringValue(sa.DisplayName)
	plan.ScopingPolicyID = nullableStringToTF(sa.ScopingPolicyId)
	plan.IsActive = types.BoolValue(sa.IsActive)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *serviceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceAccountResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.DefaultAPI.DeleteServiceAccount(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting service account", err.Error())
		return
	}
}

func (r *serviceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
