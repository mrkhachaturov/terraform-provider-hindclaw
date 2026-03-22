package provider

import (
	"context"
	"fmt"
	"net/http"

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
	_ resource.Resource                = &bankResource{}
	_ resource.ResourceWithConfigure   = &bankResource{}
	_ resource.ResourceWithImportState = &bankResource{}
)

func NewBankResource() resource.Resource {
	return &bankResource{}
}

type bankResourceModel struct {
	BankID                types.String `tfsdk:"bank_id"`
	Name                  types.String `tfsdk:"name"`
	Mission               types.String `tfsdk:"mission"`
	Background            types.String `tfsdk:"background"`
	ReflectMission        types.String `tfsdk:"reflect_mission"`
	RetainMission         types.String `tfsdk:"retain_mission"`
	EnableObservations    types.Bool   `tfsdk:"enable_observations"`
	ObservationsMission   types.String `tfsdk:"observations_mission"`
	DispositionSkepticism types.Int64  `tfsdk:"disposition_skepticism"`
	DispositionLiteralism types.Int64  `tfsdk:"disposition_literalism"`
	DispositionEmpathy    types.Int64  `tfsdk:"disposition_empathy"`
}

type bankResource struct {
	client *hindsight.APIClient
}

func (r *bankResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bank"
}

func (r *bankResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hindsight memory bank identity and profile.",
		Attributes: map[string]schema.Attribute{
			"bank_id": schema.StringAttribute{
				Description: "Bank identifier. Immutable after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Bank display name.",
				Optional:    true,
				Computed:    true,
			},
			"mission": schema.StringAttribute{
				Description: "The agent's mission.",
				Optional:    true,
				Computed:    true,
			},
			"background": schema.StringAttribute{
				Description: "Background context for the agent.",
				Optional:    true,
				Computed:    true,
			},
			"reflect_mission": schema.StringAttribute{
				Description: "Custom reflect mission. Write-only (not returned by GetBankProfile).",
				Optional:    true,
			},
			"retain_mission": schema.StringAttribute{
				Description: "Custom retain mission. Write-only (not returned by GetBankProfile).",
				Optional:    true,
			},
			"enable_observations": schema.BoolAttribute{
				Description: "Enable observations. Write-only (not returned by GetBankProfile).",
				Optional:    true,
			},
			"observations_mission": schema.StringAttribute{
				Description: "Custom observations mission. Write-only (not returned by GetBankProfile).",
				Optional:    true,
			},
			"disposition_skepticism": schema.Int64Attribute{
				Description: "Skepticism (1=trusting, 5=skeptical).",
				Optional:    true,
				Computed:    true,
			},
			"disposition_literalism": schema.Int64Attribute{
				Description: "Literalism (1=flexible, 5=literal).",
				Optional:    true,
				Computed:    true,
			},
			"disposition_empathy": schema.Int64Attribute{
				Description: "Empathy (1=detached, 5=empathetic).",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *bankResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *bankResource) buildRequest(plan *bankResourceModel) *hindsight.CreateBankRequest {
	req := hindsight.NewCreateBankRequest()
	if !plan.Name.IsNull() {
		req.SetName(plan.Name.ValueString())
	} else {
		req.SetNameNil()
	}
	if !plan.Mission.IsNull() {
		req.SetMission(plan.Mission.ValueString())
	} else {
		req.SetMissionNil()
	}
	if !plan.Background.IsNull() {
		req.SetBackground(plan.Background.ValueString())
	} else {
		req.SetBackgroundNil()
	}
	if !plan.ReflectMission.IsNull() {
		req.SetReflectMission(plan.ReflectMission.ValueString())
	} else {
		req.SetReflectMissionNil()
	}
	if !plan.RetainMission.IsNull() {
		req.SetRetainMission(plan.RetainMission.ValueString())
	} else {
		req.SetRetainMissionNil()
	}
	if !plan.EnableObservations.IsNull() {
		req.SetEnableObservations(plan.EnableObservations.ValueBool())
	} else {
		req.SetEnableObservationsNil()
	}
	if !plan.ObservationsMission.IsNull() {
		req.SetObservationsMission(plan.ObservationsMission.ValueString())
	} else {
		req.SetObservationsMissionNil()
	}
	if !plan.DispositionSkepticism.IsNull() {
		req.SetDispositionSkepticism(int32(plan.DispositionSkepticism.ValueInt64()))
	} else {
		req.SetDispositionSkepticismNil()
	}
	if !plan.DispositionLiteralism.IsNull() {
		req.SetDispositionLiteralism(int32(plan.DispositionLiteralism.ValueInt64()))
	} else {
		req.SetDispositionLiteralismNil()
	}
	if !plan.DispositionEmpathy.IsNull() {
		req.SetDispositionEmpathy(int32(plan.DispositionEmpathy.ValueInt64()))
	} else {
		req.SetDispositionEmpathyNil()
	}
	return req
}

// readProfileIntoState maps BankProfileResponse fields into the model.
// Write-only fields (reflect_mission, retain_mission, etc.) are preserved from plan/state.
//
// Hindsight may derive profile mission/background from other config-layer values
// such as reflect_mission. When Terraform explicitly manages mission/background,
// preserve those configured values instead of round-tripping the derived/effective
// API response back into state, which would otherwise cause perpetual drift.
func (r *bankResource) readProfileIntoState(profile *hindsight.BankProfileResponse, state *bankResourceModel) {
	state.BankID = types.StringValue(profile.BankId)
	state.Name = types.StringValue(profile.Name)
	if state.Mission.IsNull() || state.Mission.IsUnknown() {
		state.Mission = types.StringValue(profile.Mission)
	}
	if state.Background.IsNull() || state.Background.IsUnknown() {
		state.Background = nullableStringToTF(profile.Background)
	}
	state.DispositionSkepticism = types.Int64Value(int64(profile.Disposition.Skepticism))
	state.DispositionLiteralism = types.Int64Value(int64(profile.Disposition.Literalism))
	state.DispositionEmpathy = types.Int64Value(int64(profile.Disposition.Empathy))
	// Write-only fields not touched — caller preserves them from plan/prior state.
}

func (r *bankResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bankResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := r.buildRequest(&plan)

	profile, _, err := r.client.BanksAPI.CreateOrUpdateBank(ctx, plan.BankID.ValueString()).CreateBankRequest(*createReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating bank", err.Error())
		return
	}

	r.readProfileIntoState(profile, &plan)

	tflog.Trace(ctx, "created bank", map[string]any{"bank_id": plan.BankID.ValueString()})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *bankResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bankResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	profile, httpResp, err := r.client.BanksAPI.GetBankProfile(ctx, state.BankID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading bank", err.Error())
		return
	}

	r.readProfileIntoState(profile, &state)
	// Write-only fields preserved from prior state (already in state variable).

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *bankResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bankResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := r.buildRequest(&plan)

	profile, _, err := r.client.BanksAPI.UpdateBank(ctx, plan.BankID.ValueString()).CreateBankRequest(*updateReq).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating bank", err.Error())
		return
	}

	r.readProfileIntoState(profile, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *bankResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bankResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, httpResp, err := r.client.BanksAPI.DeleteBank(ctx, state.BankID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Error deleting bank", err.Error())
		return
	}
}

func (r *bankResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bank_id"), req, resp)
}
