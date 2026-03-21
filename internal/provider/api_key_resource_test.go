package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApiKeyResource(t *testing.T) {
	rUser := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "hindclaw_user" "test" {
  id           = %q
  display_name = "Test User"
}

resource "hindclaw_api_key" "test" {
  user_id     = hindclaw_user.test.id
  description = "acceptance test key"
}`, rUser),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hindclaw_api_key.test", "api_key"),
					resource.TestCheckResourceAttrSet("hindclaw_api_key.test", "id"),
					resource.TestCheckResourceAttr("hindclaw_api_key.test", "description", "acceptance test key"),
				),
			},
			// Verify api_key survives refresh (no import — not supported)
			{
				Config: fmt.Sprintf(`
resource "hindclaw_user" "test" {
  id           = %q
  display_name = "Test User"
}

resource "hindclaw_api_key" "test" {
  user_id     = hindclaw_user.test.id
  description = "acceptance test key"
}`, rUser),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hindclaw_api_key.test", "api_key"),
				),
			},
		},
	})
}
