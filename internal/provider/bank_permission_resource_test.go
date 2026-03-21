package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBankPermissionResource(t *testing.T) {
	rBank := acctest.RandomWithPrefix("tf-test")
	rGroup := acctest.RandomWithPrefix("tf-test")

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

resource "hindclaw_group" "test" {
  id           = %q
  display_name = "Test Group"
}

resource "hindclaw_bank_permission" "test" {
  bank_id    = hindclaw_bank.test.bank_id
  scope_type = "group"
  scope_id   = hindclaw_group.test.id
  recall     = true
  retain     = false
}`, rBank, rGroup),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_bank_permission.test", "bank_id", rBank),
					resource.TestCheckResourceAttr("hindclaw_bank_permission.test", "scope_type", "group"),
					resource.TestCheckResourceAttr("hindclaw_bank_permission.test", "recall", "true"),
					resource.TestCheckResourceAttr("hindclaw_bank_permission.test", "retain", "false"),
				),
			},
			{
				ResourceName:      "hindclaw_bank_permission.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s/group/%s", rBank, rGroup),
			},
		},
	})
}
