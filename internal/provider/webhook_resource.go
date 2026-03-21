package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	hindsight "github.com/vectorize-io/hindsight/hindsight-clients/go"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &webhookResource{}
	_ resource.ResourceWithConfigure   = &webhookResource{}
	_ resource.ResourceWithImportState = &webhookResource{}
)

func NewWebhookResource() resource.Resource {
	return &webhookResource{}
}

type webhookResourceModel struct {
	BankID     types.String `tfsdk:"bank_id"`
	ID         types.String `tfsdk:"id"`
	URL        types.String `tfsdk:"url"`
	EventTypes types.List   `tfsdk:"event_types"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	Secret     types.String `tfsdk:"secret"`
}

type webhookResource struct {
	client *hindsight.APIClient
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hindsight webhook.",
		Attributes: map[string]schema.Attribute{
			"bank_id": schema.StringAttribute{
				Description: "Bank identifier.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "Webhook identifier (computed by server).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Description: "Webhook URL.",
				Required:    true,
			},
			"event_types": schema.ListAttribute{
				Description: "Event types to subscribe to.",
				Required:    true,
				ElementType: types.StringType,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the webhook is enabled (default true).",
				Optional:    true,
				Computed:    true,
			},
			"secret": schema.StringAttribute{
				Description: "Webhook secret for signature verification. Preserved in state, may not be returned by API.",
				Optional:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := hindsight.NewCreateWebhookRequest(plan.URL.ValueString())
	var eventTypes []string
	diags = plan.EventTypes.ElementsAs(ctx, &eventTypes, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	createReq.SetEventTypes(eventTypes)

	if !plan.Enabled.IsNull() {
		createReq.SetEnabled(plan.Enabled.ValueBool())
	}
	if !plan.Secret.IsNull() {
		createReq.SetSecret(plan.Secret.ValueString())
	}

	webhook, _, err := r.client.WebhooksAPI.CreateWebhook(ctx, plan.BankID.ValueString()).CreateWebhookRequest(*createReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook", err.Error())
		return
	}

	plan.ID = types.StringValue(webhook.Id)
	plan.URL = types.StringValue(webhook.Url)
	plan.Enabled = types.BoolValue(webhook.Enabled)

	evtList, d := types.ListValueFrom(ctx, types.StringType, webhook.EventTypes)
	resp.Diagnostics.Append(d...)
	plan.EventTypes = evtList
	// Secret preserved from plan — API may not return it.

	tflog.Trace(ctx, "created webhook", map[string]any{"id": webhook.Id})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhooks, httpResp, err := r.client.WebhooksAPI.ListWebhooks(ctx, state.BankID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading webhooks", err.Error())
		return
	}

	found := false
	for _, w := range webhooks.Items {
		if w.Id == state.ID.ValueString() {
			state.URL = types.StringValue(w.Url)
			state.Enabled = types.BoolValue(w.Enabled)

			evtList, d := types.ListValueFrom(ctx, types.StringType, w.EventTypes)
			resp.Diagnostics.Append(d...)
			state.EventTypes = evtList

			// Preserve secret from prior state — API may return null/masked.
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

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := hindsight.NewUpdateWebhookRequest()
	updateReq.SetUrl(plan.URL.ValueString())

	var eventTypes []string
	diags = plan.EventTypes.ElementsAs(ctx, &eventTypes, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateReq.SetEventTypes(eventTypes)

	if !plan.Enabled.IsNull() {
		updateReq.SetEnabled(plan.Enabled.ValueBool())
	} else {
		updateReq.SetEnabledNil()
	}
	if !plan.Secret.IsNull() {
		updateReq.SetSecret(plan.Secret.ValueString())
	} else {
		updateReq.SetSecretNil()
	}

	webhook, _, err := r.client.WebhooksAPI.UpdateWebhook(ctx, plan.BankID.ValueString(), plan.ID.ValueString()).UpdateWebhookRequest(*updateReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook", err.Error())
		return
	}

	plan.URL = types.StringValue(webhook.Url)
	plan.Enabled = types.BoolValue(webhook.Enabled)

	evtList, d := types.ListValueFrom(ctx, types.StringType, webhook.EventTypes)
	resp.Diagnostics.Append(d...)
	plan.EventTypes = evtList
	// Secret preserved from plan.

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, httpResp, err := r.client.WebhooksAPI.DeleteWebhook(ctx, state.BankID.ValueString(), state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting webhook", err.Error())
		return
	}
}

func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: {bank_id}/{id}. Note: secret will be null after import.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bank_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
