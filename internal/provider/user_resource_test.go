package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserResource(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "hindclaw_user" "test" {
  id           = %q
  display_name = "TF Test User"
  email        = "test@example.com"
}`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_user.test", "id", rName),
					resource.TestCheckResourceAttr("hindclaw_user.test", "display_name", "TF Test User"),
					resource.TestCheckResourceAttr("hindclaw_user.test", "email", "test@example.com"),
				),
			},
			{
				ResourceName:      "hindclaw_user.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
resource "hindclaw_user" "test" {
  id           = %q
  display_name = "TF Test User Updated"
  email        = "test@example.com"
}`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_user.test", "display_name", "TF Test User Updated"),
				),
			},
		},
	})
}
