package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServiceAccountKeyResource(t *testing.T) {
	rUser := acctest.RandomWithPrefix("tf-test")
	rSA := acctest.RandomWithPrefix("tf-test")

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

resource "hindclaw_service_account" "test" {
  id            = %q
  owner_user_id = hindclaw_user.test.id
  display_name  = "Test SA"
}

resource "hindclaw_service_account_key" "test" {
  service_account_id = hindclaw_service_account.test.id
  description        = "acceptance test key"
}`, rUser, rSA),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hindclaw_service_account_key.test", "id"),
					resource.TestCheckResourceAttrSet("hindclaw_service_account_key.test", "api_key"),
					resource.TestCheckResourceAttr("hindclaw_service_account_key.test", "description", "acceptance test key"),
				),
			},
			// Verify api_key survives refresh
			{
				Config: fmt.Sprintf(`
resource "hindclaw_user" "test" {
  id           = %q
  display_name = "Test User"
}

resource "hindclaw_service_account" "test" {
  id            = %q
  owner_user_id = hindclaw_user.test.id
  display_name  = "Test SA"
}

resource "hindclaw_service_account_key" "test" {
  service_account_id = hindclaw_service_account.test.id
  description        = "acceptance test key"
}`, rUser, rSA),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hindclaw_service_account_key.test", "api_key"),
				),
			},
		},
	})
}
