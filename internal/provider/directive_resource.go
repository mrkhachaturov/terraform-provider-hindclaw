package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	hindsight "github.com/vectorize-io/hindsight/hindsight-clients/go"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &directiveResource{}
	_ resource.ResourceWithConfigure   = &directiveResource{}
	_ resource.ResourceWithImportState = &directiveResource{}
)

func NewDirectiveResource() resource.Resource {
	return &directiveResource{}
}

type directiveResourceModel struct {
	BankID   types.String `tfsdk:"bank_id"`
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Content  types.String `tfsdk:"content"`
	Priority types.Int64  `tfsdk:"priority"`
	IsActive types.Bool   `tfsdk:"is_active"`
	Tags     types.List   `tfsdk:"tags"`
}

type directiveResource struct {
	client *hindsight.APIClient
}

func (r *directiveResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directive"
}

func (r *directiveResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hindsight directive.",
		Attributes: map[string]schema.Attribute{
			"bank_id": schema.StringAttribute{
				Description: "Bank identifier.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "Directive identifier (computed by server).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Directive name.",
				Required:    true,
			},
			"content": schema.StringAttribute{
				Description: "Directive content.",
				Required:    true,
			},
			"priority": schema.Int64Attribute{
				Description: "Priority (default 0).",
				Optional:    true,
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether the directive is active (default true).",
				Optional:    true,
			},
			"tags": schema.ListAttribute{
				Description: "Tags.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *directiveResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *directiveResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan directiveResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := hindsight.NewCreateDirectiveRequest(plan.Name.ValueString(), plan.Content.ValueString())
	if !plan.Priority.IsNull() {
		createReq.SetPriority(int32(plan.Priority.ValueInt64()))
	}
	if !plan.IsActive.IsNull() {
		createReq.SetIsActive(plan.IsActive.ValueBool())
	}
	if !plan.Tags.IsNull() {
		var tags []string
		diags = plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		createReq.SetTags(tags)
	}

	directive, _, err := r.client.DirectivesAPI.CreateDirective(ctx, plan.BankID.ValueString()).CreateDirectiveRequest(*createReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating directive", err.Error())
		return
	}

	r.readResponseIntoState(ctx, directive, &plan, &resp.Diagnostics)

	tflog.Trace(ctx, "created directive", map[string]any{"id": directive.Id})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *directiveResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state directiveResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	directive, httpResp, err := r.client.DirectivesAPI.GetDirective(ctx, state.BankID.ValueString(), state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading directive", err.Error())
		return
	}

	r.readResponseIntoState(ctx, directive, &state, &resp.Diagnostics)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *directiveResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan directiveResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := hindsight.NewUpdateDirectiveRequest()
	updateReq.SetName(plan.Name.ValueString())
	updateReq.SetContent(plan.Content.ValueString())
	if !plan.Priority.IsNull() {
		updateReq.SetPriority(int32(plan.Priority.ValueInt64()))
	} else {
		updateReq.SetPriorityNil()
	}
	if !plan.IsActive.IsNull() {
		updateReq.SetIsActive(plan.IsActive.ValueBool())
	} else {
		updateReq.SetIsActiveNil()
	}
	if !plan.Tags.IsNull() {
		var tags []string
		diags = plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		updateReq.SetTags(tags)
	} else {
		updateReq.SetTags([]string{})
	}

	directive, _, err := r.client.DirectivesAPI.UpdateDirective(ctx, plan.BankID.ValueString(), plan.ID.ValueString()).UpdateDirectiveRequest(*updateReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating directive", err.Error())
		return
	}

	r.readResponseIntoState(ctx, directive, &plan, &resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *directiveResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state directiveResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, httpResp, err := r.client.DirectivesAPI.DeleteDirective(ctx, state.BankID.ValueString(), state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting directive", err.Error())
		return
	}
}

func (r *directiveResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: {bank_id}/{id}")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bank_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *directiveResource) readResponseIntoState(ctx context.Context, directive *hindsight.DirectiveResponse, state *directiveResourceModel, diags *diag.Diagnostics) {
	state.BankID = types.StringValue(directive.BankId)
	state.ID = types.StringValue(directive.Id)
	state.Name = types.StringValue(directive.Name)
	state.Content = types.StringValue(directive.Content)

	if directive.Priority != nil {
		state.Priority = types.Int64Value(int64(*directive.Priority))
	} else {
		state.Priority = types.Int64Null()
	}
	if directive.IsActive != nil {
		state.IsActive = types.BoolValue(*directive.IsActive)
	} else {
		state.IsActive = types.BoolNull()
	}
	if directive.Tags != nil {
		listVal, d := types.ListValueFrom(ctx, types.StringType, directive.Tags)
		diags.Append(d...)
		state.Tags = listVal
	} else {
		state.Tags = types.ListNull(types.StringType)
	}
}
