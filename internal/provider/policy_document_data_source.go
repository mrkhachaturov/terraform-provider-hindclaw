package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &policyDocumentDataSource{}

func NewPolicyDocumentDataSource() datasource.DataSource {
	return &policyDocumentDataSource{}
}

type policyDocumentDataSourceModel struct {
	Statements []policyDocumentStatement `tfsdk:"statement"`
	JSON       types.String              `tfsdk:"json"`
}

type policyDocumentStatement struct {
	Effect            types.String `tfsdk:"effect"`
	Actions           types.List   `tfsdk:"actions"`
	Banks             types.List   `tfsdk:"banks"`
	RecallBudget      types.String `tfsdk:"recall_budget"`
	RecallMaxTokens   types.Int64  `tfsdk:"recall_max_tokens"`
	RecallTagGroups   types.String `tfsdk:"recall_tag_groups"`
	RetainRoles       types.List   `tfsdk:"retain_roles"`
	RetainTags        types.List   `tfsdk:"retain_tags"`
	RetainEveryNTurns types.Int64  `tfsdk:"retain_every_n_turns"`
	RetainStrategy    types.String `tfsdk:"retain_strategy"`
	LlmModel          types.String `tfsdk:"llm_model"`
	LlmProvider       types.String `tfsdk:"llm_provider"`
	ExcludeProviders  types.List   `tfsdk:"exclude_providers"`
}

type policyDocumentDataSource struct{}

func (d *policyDocumentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_document"
}

func (d *policyDocumentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Builds a Hindclaw policy document JSON string from HCL statement blocks.",
		Blocks: map[string]schema.Block{
			"statement": schema.ListNestedBlock{
				Description: "Policy statement.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"effect": schema.StringAttribute{
							Description: "Statement effect: allow or deny.",
							Required:    true,
						},
						"actions": schema.ListAttribute{
							Description: "Actions this statement applies to (e.g. recall, retain).",
							Optional:    true,
							ElementType: types.StringType,
						},
						"banks": schema.ListAttribute{
							Description: "Bank IDs this statement applies to.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"recall_budget": schema.StringAttribute{
							Description: "Recall budget level.",
							Optional:    true,
						},
						"recall_max_tokens": schema.Int64Attribute{
							Description: "Maximum recall tokens.",
							Optional:    true,
						},
						"recall_tag_groups": schema.StringAttribute{
							Description: "Recall tag group filter as JSON string.",
							Optional:    true,
						},
						"retain_roles": schema.ListAttribute{
							Description: "Retain roles.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"retain_tags": schema.ListAttribute{
							Description: "Retain tags.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"retain_every_n_turns": schema.Int64Attribute{
							Description: "Retain frequency (every N turns).",
							Optional:    true,
						},
						"retain_strategy": schema.StringAttribute{
							Description: "Retain strategy name.",
							Optional:    true,
						},
						"llm_model": schema.StringAttribute{
							Description: "LLM model override.",
							Optional:    true,
						},
						"llm_provider": schema.StringAttribute{
							Description: "LLM provider override.",
							Optional:    true,
						},
						"exclude_providers": schema.ListAttribute{
							Description: "Providers to exclude from recall.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"json": schema.StringAttribute{
				Description: "The generated policy document as a JSON string.",
				Computed:    true,
			},
		},
	}
}

func (d *policyDocumentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model policyDocumentDataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	statements := make([]map[string]interface{}, 0, len(model.Statements))
	for _, s := range model.Statements {
		stmt := map[string]interface{}{
			"effect": s.Effect.ValueString(),
		}

		if !s.Actions.IsNull() {
			var actions []string
			diags = s.Actions.ElementsAs(ctx, &actions, false)
			resp.Diagnostics.Append(diags...)
			stmt["actions"] = actions
		}

		if !s.Banks.IsNull() {
			var banks []string
			diags = s.Banks.ElementsAs(ctx, &banks, false)
			resp.Diagnostics.Append(diags...)
			stmt["banks"] = banks
		}

		if !s.RecallBudget.IsNull() {
			stmt["recall_budget"] = s.RecallBudget.ValueString()
		}

		if !s.RecallMaxTokens.IsNull() {
			stmt["recall_max_tokens"] = s.RecallMaxTokens.ValueInt64()
		}

		if !s.RecallTagGroups.IsNull() {
			var tagGroups interface{}
			if err := json.Unmarshal([]byte(s.RecallTagGroups.ValueString()), &tagGroups); err != nil {
				resp.Diagnostics.AddError("Invalid recall_tag_groups JSON", err.Error())
				return
			}
			stmt["recall_tag_groups"] = tagGroups
		}

		if !s.RetainRoles.IsNull() {
			var roles []string
			diags = s.RetainRoles.ElementsAs(ctx, &roles, false)
			resp.Diagnostics.Append(diags...)
			stmt["retain_roles"] = roles
		}

		if !s.RetainTags.IsNull() {
			var tags []string
			diags = s.RetainTags.ElementsAs(ctx, &tags, false)
			resp.Diagnostics.Append(diags...)
			stmt["retain_tags"] = tags
		}

		if !s.RetainEveryNTurns.IsNull() {
			stmt["retain_every_n_turns"] = s.RetainEveryNTurns.ValueInt64()
		}

		if !s.RetainStrategy.IsNull() {
			stmt["retain_strategy"] = s.RetainStrategy.ValueString()
		}

		if !s.LlmModel.IsNull() {
			stmt["llm_model"] = s.LlmModel.ValueString()
		}

		if !s.LlmProvider.IsNull() {
			stmt["llm_provider"] = s.LlmProvider.ValueString()
		}

		if !s.ExcludeProviders.IsNull() {
			var providers []string
			diags = s.ExcludeProviders.ElementsAs(ctx, &providers, false)
			resp.Diagnostics.Append(diags...)
			stmt["exclude_providers"] = providers
		}

		statements = append(statements, stmt)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	doc := map[string]interface{}{
		"version":    "2026-03-24",
		"statements": statements,
	}

	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling policy document", err.Error())
		return
	}

	model.JSON = types.StringValue(string(jsonBytes))

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}
