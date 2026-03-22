package provider

import (
	"context"
	"encoding/json"
	"fmt"

	hindclaw "github.com/mrkhachaturov/hindclaw/hindclaw-clients/go"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &resolvedPermissionsDataSource{}
	_ datasource.DataSourceWithConfigure = &resolvedPermissionsDataSource{}
)

func NewResolvedPermissionsDataSource() datasource.DataSource {
	return &resolvedPermissionsDataSource{}
}

type resolvedPermissionsDataSourceModel struct {
	// Input
	Bank    types.String `tfsdk:"bank"`
	Sender  types.String `tfsdk:"sender"`
	Agent   types.String `tfsdk:"agent"`
	Channel types.String `tfsdk:"channel"`
	Topic   types.String `tfsdk:"topic"`
	// Computed
	UserID            types.String `tfsdk:"user_id"`
	IsAnonymous       types.Bool   `tfsdk:"is_anonymous"`
	Recall            types.Bool   `tfsdk:"recall"`
	Retain            types.Bool   `tfsdk:"retain"`
	RetainRoles       types.List   `tfsdk:"retain_roles"`
	RetainTags        types.List   `tfsdk:"retain_tags"`
	RetainEveryNTurns types.Int64  `tfsdk:"retain_every_n_turns"`
	RetainStrategy    types.String `tfsdk:"retain_strategy"`
	RecallBudget      types.String `tfsdk:"recall_budget"`
	RecallMaxTokens   types.Int64  `tfsdk:"recall_max_tokens"`
	RecallTagGroups   types.String `tfsdk:"recall_tag_groups"`
	LlmModel          types.String `tfsdk:"llm_model"`
	LlmProvider       types.String `tfsdk:"llm_provider"`
	ExcludeProviders  types.List   `tfsdk:"exclude_providers"`
}

type resolvedPermissionsDataSource struct {
	client *hindclaw.APIClient
}

func (d *resolvedPermissionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resolved_permissions"
}

func (d *resolvedPermissionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resolves effective permissions for a given bank and optional sender/agent/channel/topic context.",
		Attributes: map[string]schema.Attribute{
			"bank":    schema.StringAttribute{Description: "Bank ID.", Required: true},
			"sender":  schema.StringAttribute{Description: "Sender (e.g. telegram:100001).", Optional: true},
			"agent":   schema.StringAttribute{Description: "Agent name.", Optional: true},
			"channel": schema.StringAttribute{Description: "Channel name.", Optional: true},
			"topic":   schema.StringAttribute{Description: "Topic ID.", Optional: true},
			// Computed
			"user_id":              schema.StringAttribute{Description: "Resolved user ID.", Computed: true},
			"is_anonymous":         schema.BoolAttribute{Description: "Whether the sender is anonymous.", Computed: true},
			"recall":               schema.BoolAttribute{Description: "Effective recall permission.", Computed: true},
			"retain":               schema.BoolAttribute{Description: "Effective retain permission.", Computed: true},
			"retain_roles":         schema.ListAttribute{Description: "Effective retain roles.", Computed: true, ElementType: types.StringType},
			"retain_tags":          schema.ListAttribute{Description: "Effective retain tags.", Computed: true, ElementType: types.StringType},
			"retain_every_n_turns": schema.Int64Attribute{Description: "Effective retain frequency.", Computed: true},
			"retain_strategy":      schema.StringAttribute{Description: "Effective retain strategy.", Computed: true},
			"recall_budget":        schema.StringAttribute{Description: "Effective recall budget.", Computed: true},
			"recall_max_tokens":    schema.Int64Attribute{Description: "Effective max recall tokens.", Computed: true},
			"recall_tag_groups":    schema.StringAttribute{Description: "Effective tag group filter (JSON).", Computed: true},
			"llm_model":            schema.StringAttribute{Description: "Effective LLM model.", Computed: true},
			"llm_provider":         schema.StringAttribute{Description: "Effective LLM provider.", Computed: true},
			"exclude_providers":    schema.ListAttribute{Description: "Effective excluded providers.", Computed: true, ElementType: types.StringType},
		},
	}
}

func (d *resolvedPermissionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*hindclawClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *hindclawClients, got: %T", req.ProviderData),
		)
		return
	}
	d.client = clients.hindclaw
}

func (d *resolvedPermissionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config resolvedPermissionsDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := d.client.DefaultAPI.DebugResolve(ctx).Bank(config.Bank.ValueString())
	if !config.Sender.IsNull() {
		apiReq = apiReq.Sender(config.Sender.ValueString())
	}
	if !config.Agent.IsNull() {
		apiReq = apiReq.Agent(config.Agent.ValueString())
	}
	if !config.Channel.IsNull() {
		apiReq = apiReq.Channel(config.Channel.ValueString())
	}
	if !config.Topic.IsNull() {
		apiReq = apiReq.Topic(config.Topic.ValueString())
	}

	resolved, _, err := apiReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error resolving permissions", err.Error())
		return
	}

	config.UserID = types.StringValue(resolved.UserId)
	config.IsAnonymous = types.BoolValue(resolved.IsAnonymous)
	config.Recall = types.BoolValue(resolved.Recall)
	config.Retain = types.BoolValue(resolved.Retain)
	config.RetainEveryNTurns = types.Int64Value(int64(resolved.RetainEveryNTurns))
	config.RecallBudget = types.StringValue(resolved.RecallBudget)
	config.RecallMaxTokens = types.Int64Value(int64(resolved.RecallMaxTokens))

	config.RetainStrategy = nullableStringToTF(resolved.RetainStrategy)
	config.LlmModel = nullableStringToTF(resolved.LlmModel)
	config.LlmProvider = nullableStringToTF(resolved.LlmProvider)

	retainRoles, diags2 := types.ListValueFrom(ctx, types.StringType, resolved.RetainRoles)
	resp.Diagnostics.Append(diags2...)
	config.RetainRoles = retainRoles

	retainTags, diags2 := types.ListValueFrom(ctx, types.StringType, resolved.RetainTags)
	resp.Diagnostics.Append(diags2...)
	config.RetainTags = retainTags

	excludeProviders, diags2 := types.ListValueFrom(ctx, types.StringType, resolved.ExcludeProviders)
	resp.Diagnostics.Append(diags2...)
	config.ExcludeProviders = excludeProviders

	if resolved.RecallTagGroups != nil {
		jsonBytes, err := json.Marshal(resolved.RecallTagGroups)
		if err != nil {
			resp.Diagnostics.AddError("Error marshaling recall_tag_groups", err.Error())
			return
		}
		config.RecallTagGroups = types.StringValue(string(jsonBytes))
	} else {
		config.RecallTagGroups = types.StringNull()
	}

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
