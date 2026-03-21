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
	_ resource.Resource                = &userChannelResource{}
	_ resource.ResourceWithConfigure   = &userChannelResource{}
	_ resource.ResourceWithImportState = &userChannelResource{}
)

func NewUserChannelResource() resource.Resource {
	return &userChannelResource{}
}

type userChannelResourceModel struct {
	UserID          types.String `tfsdk:"user_id"`
	ChannelProvider types.String `tfsdk:"channel_provider"`
	SenderID        types.String `tfsdk:"sender_id"`
}

type userChannelResource struct {
	client *hindclaw.APIClient
}

func (r *userChannelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_channel"
}

func (r *userChannelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Maps a channel sender to a Hindclaw user.",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Description: "User identifier.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"channel_provider": schema.StringAttribute{
				Description: "Channel provider (e.g. telegram).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"sender_id": schema.StringAttribute{
				Description: "Sender identifier within the channel.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *userChannelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userChannelResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	addReq := hindclaw.NewAddChannelRequest(plan.ChannelProvider.ValueString(), plan.SenderID.ValueString())

	_, _, err := r.client.DefaultAPI.AddUserChannel(ctx, plan.UserID.ValueString()).AddChannelRequest(*addReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating user channel", err.Error())
		return
	}

	tflog.Trace(ctx, "created user channel", map[string]any{
		"user_id":  plan.UserID.ValueString(),
		"provider": plan.ChannelProvider.ValueString(),
		"sender":   plan.SenderID.ValueString(),
	})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *userChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userChannelResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	channels, httpResp, err := r.client.DefaultAPI.ListUserChannels(ctx, state.UserID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user channels", err.Error())
		return
	}

	// Find matching channel by provider + sender_id
	found := false
	for _, ch := range channels {
		if ch.Provider == state.ChannelProvider.ValueString() && ch.SenderId == state.SenderID.ValueString() {
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

func (r *userChannelResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "All fields use RequiresReplace — update should never be called.")
}

func (r *userChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userChannelResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DefaultAPI.RemoveUserChannel(ctx,
		state.UserID.ValueString(),
		state.ChannelProvider.ValueString(),
		state.SenderID.ValueString(),
	).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error deleting user channel", err.Error())
		return
	}
}

func (r *userChannelResource) ImportState(_ context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: {user_id}/{channel_provider}/{sender_id}")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(context.Background(), path.Root("user_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(context.Background(), path.Root("channel_provider"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(context.Background(), path.Root("sender_id"), parts[2])...)
}
