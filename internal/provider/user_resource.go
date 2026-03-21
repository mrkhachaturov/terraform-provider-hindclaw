package provider

import (
	"context"
	"fmt"

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
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

func NewUserResource() resource.Resource {
	return &userResource{}
}

type userResourceModel struct {
	ID          types.String `tfsdk:"id"`
	DisplayName types.String `tfsdk:"display_name"`
	Email       types.String `tfsdk:"email"`
}

type userResource struct {
	client *hindclaw.APIClient
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hindclaw user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "User identifier. Immutable after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "Display name.",
				Required:    true,
			},
			"email": schema.StringAttribute{
				Description: "Email address.",
				Optional:    true,
			},
		},
	}
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := hindclaw.NewCreateUserRequest(plan.ID.ValueString(), plan.DisplayName.ValueString())
	if !plan.Email.IsNull() {
		createReq.SetEmail(plan.Email.ValueString())
	}

	user, _, err := r.client.DefaultAPI.CreateUser(ctx).CreateUserRequest(*createReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}

	plan.ID = types.StringValue(user.Id)
	plan.DisplayName = types.StringValue(user.DisplayName)
	if user.Email.IsSet() {
		plan.Email = types.StringValue(*user.Email.Get())
	}

	tflog.Trace(ctx, "created user", map[string]any{"id": user.Id})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, _, err := r.client.DefaultAPI.GetUser(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	state.ID = types.StringValue(user.Id)
	state.DisplayName = types.StringValue(user.DisplayName)
	if user.Email.IsSet() {
		state.Email = types.StringValue(*user.Email.Get())
	} else {
		state.Email = types.StringNull()
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := hindclaw.NewUpdateUserRequest()
	updateReq.SetDisplayName(plan.DisplayName.ValueString())
	if !plan.Email.IsNull() {
		updateReq.SetEmail(plan.Email.ValueString())
	} else {
		// Explicitly send null to clear the field remotely.
		// Without this, removing email from Terraform config would never
		// clear it on the server (API treats omitted fields as "leave unchanged").
		updateReq.SetEmailNil()
	}

	user, _, err := r.client.DefaultAPI.UpdateUser(ctx, plan.ID.ValueString()).UpdateUserRequest(*updateReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}

	plan.ID = types.StringValue(user.Id)
	plan.DisplayName = types.StringValue(user.DisplayName)
	if user.Email.IsSet() {
		plan.Email = types.StringValue(*user.Email.Get())
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DefaultAPI.DeleteUser(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error deleting user", err.Error())
		return
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
