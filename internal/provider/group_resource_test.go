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
  recall       = true
  retain       = false
}`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_group.test", "id", rName),
					resource.TestCheckResourceAttr("hindclaw_group.test", "display_name", "TF Test Group"),
					resource.TestCheckResourceAttr("hindclaw_group.test", "recall", "true"),
					resource.TestCheckResourceAttr("hindclaw_group.test", "retain", "false"),
				),
			},
			{
				ResourceName:      "hindclaw_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
resource "hindclaw_group" "test" {
  id           = %q
  display_name = "TF Test Group Updated"
  recall       = false
}`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_group.test", "display_name", "TF Test Group Updated"),
					resource.TestCheckResourceAttr("hindclaw_group.test", "recall", "false"),
				),
			},
		},
	})
}
