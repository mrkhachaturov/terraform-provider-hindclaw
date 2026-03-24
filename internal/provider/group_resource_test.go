package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupResource(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "hindclaw_group" "test" {
  id           = %q
  display_name = "TF Test Group"
}`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_group.test", "id", rName),
					resource.TestCheckResourceAttr("hindclaw_group.test", "display_name", "TF Test Group"),
					resource.TestCheckResourceAttr("hindclaw_group.test", "force_destroy", "false"),
				),
			},
			// Import
			{
				ResourceName:            "hindclaw_group.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_destroy"},
			},
			// Update display_name
			{
				Config: fmt.Sprintf(`
resource "hindclaw_group" "test" {
  id           = %q
  display_name = "TF Test Group Updated"
}`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_group.test", "display_name", "TF Test Group Updated"),
				),
			},
		},
	})
}
