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
					resource.TestCheckResourceAttr("hindclaw_user.test", "disable_user", "false"),
					resource.TestCheckResourceAttr("hindclaw_user.test", "force_destroy", "false"),
				),
			},
			// Import
			{
				ResourceName:            "hindclaw_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_destroy"},
			},
			// Update display_name
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
			// Disable user
			{
				Config: fmt.Sprintf(`
resource "hindclaw_user" "test" {
  id           = %q
  display_name = "TF Test User Updated"
  email        = "test@example.com"
  disable_user = true
}`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_user.test", "disable_user", "true"),
				),
			},
			// Re-enable user
			{
				Config: fmt.Sprintf(`
resource "hindclaw_user" "test" {
  id           = %q
  display_name = "TF Test User Updated"
  email        = "test@example.com"
  disable_user = false
}`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_user.test", "disable_user", "false"),
				),
			},
		},
	})
}
