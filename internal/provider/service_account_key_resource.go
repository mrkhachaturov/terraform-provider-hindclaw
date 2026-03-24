package provider

import (
	"context"
	"fmt"
	"net/http"

	hindclaw "github.com/mrkhachaturov/hindclaw/hindclaw-clients/go"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = &serviceAccountKeyResource{}
	_ resource.ResourceWithConfigure = &serviceAccountKeyResource{}
)

func NewServiceAccountKeyResource() resource.Resource {
	return &serviceAccountKeyResource{}
}

type serviceAccountKeyResourceModel struct {
	ServiceAccountID types.String `tfsdk:"service_account_id"`
	Description      types.String `tfsdk:"description"`
	ID               types.String `tfsdk:"id"`
	ApiKey           types.String `tfsdk:"api_key"`
}

type serviceAccountKeyResource struct {
	client *hindclaw.APIClient
}

func (r *serviceAccountKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_key"
}

func (r *serviceAccountKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hindclaw service account API key. The full key is only available at creation time.",
		Attributes: map[string]schema.Attribute{
			"service_account_id": schema.StringAttribute{
				Description: "Service account identifier.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Key description. Changing forces recreation.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "Key identifier (computed by server).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"api_key": schema.StringAttribute{
				Description: "Full API key (only available at creation, stored in state).",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *serviceAccountKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serviceAccountKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceAccountKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := hindclaw.NewCreateSAKeyRequest()
	if !plan.Description.IsNull() {
		createReq.SetDescription(plan.Description.ValueString())
	}

	key, _, err := r.client.DefaultAPI.CreateSaKey(ctx, plan.ServiceAccountID.ValueString()).CreateSAKeyRequest(*createReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating service account key", err.Error())
		return
	}

	plan.ID = types.StringValue(key.Id)
	plan.ApiKey = types.StringValue(key.ApiKey)
	plan.Description = nullableStringToTF(key.Description)

	tflog.Trace(ctx, "created service account key", map[string]any{"id": key.Id})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *serviceAccountKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceAccountKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	keys, httpResp, err := r.client.DefaultAPI.ListSaKeys(ctx, state.ServiceAccountID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading service account keys", err.Error())
		return
	}

	found := false
	for _, k := range keys {
		if k.Id == state.ID.ValueString() {
			state.Description = nullableStringToTF(k.Description)
			found = true
			break
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	// Preserve api_key from state — not returned by list endpoint
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *serviceAccountKeyResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Service account key updates are not supported. Delete and recreate.")
}

func (r *serviceAccountKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceAccountKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.DefaultAPI.DeleteSaKey(ctx,
		state.ServiceAccountID.ValueString(),
		state.ID.ValueString(),
	).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting service account key", err.Error())
		return
	}
}
