package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPolicyResource(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: fmt.Sprintf(`
resource "hindclaw_policy" "test" {
  id           = %q
  display_name = "TF Test Policy"
  document     = jsonencode({
    version = "2026-03-24"
    statements = [{
      effect  = "allow"
      actions = ["bank:recall", "bank:reflect"]
      banks   = ["*"]
      recall_budget = "mid"
    }]
  })
}`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_policy.test", "id", rName),
					resource.TestCheckResourceAttr("hindclaw_policy.test", "display_name", "TF Test Policy"),
					resource.TestCheckResourceAttrSet("hindclaw_policy.test", "document"),
				),
			},
			// Import
			{
				ResourceName:      "hindclaw_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: fmt.Sprintf(`
resource "hindclaw_policy" "test" {
  id           = %q
  display_name = "TF Test Policy Updated"
  document     = jsonencode({
    version = "2026-03-24"
    statements = [{
      effect  = "allow"
      actions = ["bank:recall", "bank:reflect"]
      banks   = ["*"]
      recall_budget = "high"
    }]
  })
}`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_policy.test", "display_name", "TF Test Policy Updated"),
				),
			},
		},
	})
}
