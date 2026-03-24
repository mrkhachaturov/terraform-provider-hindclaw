package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPolicyDocumentDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "hindclaw_policy_document" "test" {
  statement {
    effect  = "allow"
    actions = ["bank:recall", "bank:reflect"]
    banks   = ["*"]
    recall_budget     = "mid"
    recall_max_tokens = 1024
  }
  statement {
    effect  = "deny"
    actions = ["bank:retain"]
    banks   = ["bb9e"]
  }
}

output "policy_json" {
  value = data.hindclaw_policy_document.test.json
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.hindclaw_policy_document.test", "json"),
				),
			},
		},
	})
}

func TestAccPolicyDocumentDataSource_AllFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "hindclaw_policy_document" "full" {
  statement {
    effect               = "allow"
    actions              = ["bank:recall", "bank:reflect", "bank:retain"]
    banks                = ["yoda", "r2d2"]
    recall_budget        = "high"
    recall_max_tokens    = 2048
    retain_roles         = ["user", "assistant"]
    retain_tags          = ["executive", "priority"]
    retain_every_n_turns = 1
    retain_strategy      = "yoda-thorough"
    llm_model            = "claude-sonnet-4-6"
    llm_provider         = "anthropic"
    exclude_providers    = ["web"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.hindclaw_policy_document.full", "json"),
				),
			},
		},
	})
}
