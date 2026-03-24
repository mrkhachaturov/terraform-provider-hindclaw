package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBankPolicyResource(t *testing.T) {
	// Use an existing bank (yoda) — hindclaw_bank creation uses the Hindsight
	// native API which has a client/API version mismatch for the "input" field.
	// Bank policy CRUD is what we're testing here, not bank creation.
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "hindclaw_bank_policy" "test" {
  bank_id = "yoda"
  document = jsonencode({
    version          = "2026-03-24"
    default_strategy = "yoda-acc-test"
  })
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_bank_policy.test", "bank_id", "yoda"),
					resource.TestCheckResourceAttrSet("hindclaw_bank_policy.test", "document"),
				),
			},
			// Import
			{
				ResourceName:                         "hindclaw_bank_policy.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bank_id",
				ImportStateId:                        "yoda",
			},
			// Update document
			{
				Config: `
resource "hindclaw_bank_policy" "test" {
  bank_id = "yoda"
  document = jsonencode({
    version          = "2026-03-24"
    default_strategy = "yoda-acc-test-updated"
    strategy_overrides = [
      { scope = "channel", value = "telegram", strategy = "yoda-telegram" }
    ]
  })
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hindclaw_bank_policy.test", "document"),
				),
			},
		},
	})
}
