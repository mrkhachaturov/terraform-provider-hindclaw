package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBankPolicyResource(t *testing.T) {
	rBank := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "hindclaw_bank" "test" {
  bank_id = %q
  name    = "Test Bank"
}

resource "hindclaw_bank_policy" "test" {
  bank_id = hindclaw_bank.test.bank_id
  document = jsonencode({
    version          = "2026-03-24"
    default_strategy = "test-default"
  })
}`, rBank),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_bank_policy.test", "bank_id", rBank),
					resource.TestCheckResourceAttrSet("hindclaw_bank_policy.test", "document"),
				),
			},
			// Import
			{
				ResourceName:      "hindclaw_bank_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update document
			{
				Config: fmt.Sprintf(`
resource "hindclaw_bank" "test" {
  bank_id = %q
  name    = "Test Bank"
}

resource "hindclaw_bank_policy" "test" {
  bank_id = hindclaw_bank.test.bank_id
  document = jsonencode({
    version          = "2026-03-24"
    default_strategy = "test-updated"
    strategy_overrides = [
      { scope = "channel", value = "telegram", strategy = "test-telegram" }
    ]
  })
}`, rBank),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hindclaw_bank_policy.test", "document"),
				),
			},
		},
	})
}
