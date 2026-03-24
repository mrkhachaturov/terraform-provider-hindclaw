package provider

import (
	"context"
	"fmt"
	"net/http"

	hindclaw "github.com/mrkhachaturov/hindclaw/hindclaw-clients/go"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	ID           types.String `tfsdk:"id"`
	DisplayName  types.String `tfsdk:"display_name"`
	Email        types.String `tfsdk:"email"`
	DisableUser  types.Bool   `tfsdk:"disable_user"`
	ForceDestroy types.Bool   `tfsdk:"force_destroy"`
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
			"disable_user": schema.BoolAttribute{
				Description: "When true, user is disabled (maps to is_active=false on the API).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"force_destroy": schema.BoolAttribute{
				Description: "When true, deleting the user cascades to service accounts, API keys, policy attachments, group memberships, and channel mappings.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
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
	if plan.DisableUser.ValueBool() {
		createReq.SetIsActive(false)
	}

	user, _, err := r.client.DefaultAPI.CreateUser(ctx).CreateUserRequest(*createReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}

	plan.ID = types.StringValue(user.Id)
	plan.DisplayName = types.StringValue(user.DisplayName)
	plan.Email = nullableStringToTF(user.Email)
	plan.DisableUser = types.BoolValue(!user.GetIsActive())

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

	user, httpResp, err := r.client.DefaultAPI.GetUser(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	state.ID = types.StringValue(user.Id)
	state.DisplayName = types.StringValue(user.DisplayName)
	state.Email = nullableStringToTF(user.Email)
	state.DisableUser = types.BoolValue(!user.GetIsActive())
	// force_destroy is local-only — not returned by the API.

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
		updateReq.SetEmailNil()
	}
	// disable_user maps to is_active (inverted)
	updateReq.SetIsActive(!plan.DisableUser.ValueBool())

	user, _, err := r.client.DefaultAPI.UpdateUser(ctx, plan.ID.ValueString()).UpdateUserRequest(*updateReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}

	plan.ID = types.StringValue(user.Id)
	plan.DisplayName = types.StringValue(user.DisplayName)
	plan.Email = nullableStringToTF(user.Email)
	plan.DisableUser = types.BoolValue(!user.GetIsActive())

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

	userID := state.ID.ValueString()

	// Clean up policy attachments for this user. The policy_attachments table
	// uses a TEXT principal_id with no FK constraint, so orphaned rows would
	// remain if we deleted the user without removing them first.
	// ListPolicyAttachments is per-policy, so list all policies and filter.
	policies, _, err := r.client.DefaultAPI.ListPolicies(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error listing policies for user cleanup", err.Error())
		return
	}
	for _, p := range policies {
		attachments, _, err := r.client.DefaultAPI.ListPolicyAttachments(ctx).PolicyId(p.Id).Execute()
		if err != nil {
			resp.Diagnostics.AddError("Error listing policy attachments for user", err.Error())
			return
		}
		for _, a := range attachments {
			if a.PrincipalType == "user" && a.PrincipalId == userID {
				_, err := r.client.DefaultAPI.DeletePolicyAttachment(ctx, a.PolicyId, "user", userID).Execute()
				if err != nil {
					resp.Diagnostics.AddError("Error deleting policy attachment", err.Error())
					return
				}
			}
		}
	}

	if state.ForceDestroy.ValueBool() {
		// Delete all service accounts owned by this user (and their SA keys).
		serviceAccounts, _, err := r.client.DefaultAPI.ListServiceAccounts(ctx).Execute()
		if err != nil {
			resp.Diagnostics.AddError("Error listing service accounts for user cleanup", err.Error())
			return
		}
		for _, sa := range serviceAccounts {
			if sa.OwnerUserId != userID {
				continue
			}
			saKeys, _, err := r.client.DefaultAPI.ListSaKeys(ctx, sa.Id).Execute()
			if err != nil {
				resp.Diagnostics.AddError("Error listing SA keys for user cleanup", err.Error())
				return
			}
			for _, k := range saKeys {
				_, err := r.client.DefaultAPI.DeleteSaKey(ctx, sa.Id, k.Id).Execute()
				if err != nil {
					resp.Diagnostics.AddError("Error deleting SA key", err.Error())
					return
				}
			}
			httpResp, err := r.client.DefaultAPI.DeleteServiceAccount(ctx, sa.Id).Execute()
			if err != nil {
				if httpResp == nil || httpResp.StatusCode != http.StatusNotFound {
					resp.Diagnostics.AddError("Error deleting service account", err.Error())
					return
				}
			}
		}

		// Delete all API keys belonging to this user.
		apiKeys, _, err := r.client.DefaultAPI.ListApiKeys(ctx, userID).Execute()
		if err != nil {
			resp.Diagnostics.AddError("Error listing API keys for user cleanup", err.Error())
			return
		}
		for _, k := range apiKeys {
			_, err := r.client.DefaultAPI.DeleteApiKey(ctx, userID, k.Id).Execute()
			if err != nil {
				resp.Diagnostics.AddError("Error deleting API key", err.Error())
				return
			}
		}

		// Remove this user from all groups.
		groups, _, err := r.client.DefaultAPI.ListGroups(ctx).Execute()
		if err != nil {
			resp.Diagnostics.AddError("Error listing groups for user cleanup", err.Error())
			return
		}
		for _, g := range groups {
			members, _, err := r.client.DefaultAPI.ListGroupMembers(ctx, g.Id).Execute()
			if err != nil {
				resp.Diagnostics.AddError("Error listing group members for user cleanup", err.Error())
				return
			}
			for _, m := range members {
				if m.UserId == userID {
					_, err := r.client.DefaultAPI.RemoveGroupMember(ctx, g.Id, userID).Execute()
					if err != nil {
						resp.Diagnostics.AddError("Error removing user from group", err.Error())
						return
					}
					break
				}
			}
		}

		// Remove all channel mappings for this user.
		channels, _, err := r.client.DefaultAPI.ListUserChannels(ctx, userID).Execute()
		if err != nil {
			resp.Diagnostics.AddError("Error listing user channels for user cleanup", err.Error())
			return
		}
		for _, ch := range channels {
			_, err := r.client.DefaultAPI.RemoveUserChannel(ctx, userID, ch.Provider, ch.SenderId).Execute()
			if err != nil {
				resp.Diagnostics.AddError("Error removing user channel", err.Error())
				return
			}
		}
	}

	httpResp, err := r.client.DefaultAPI.DeleteUser(ctx, userID).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting user", err.Error())
		return
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
