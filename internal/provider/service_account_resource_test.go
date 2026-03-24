package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServiceAccountResource(t *testing.T) {
	rUser := acctest.RandomWithPrefix("tf-test")
	rSA := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without scoping policy
			{
				Config: fmt.Sprintf(`
resource "hindclaw_user" "test" {
  id           = %q
  display_name = "Test User"
}

resource "hindclaw_service_account" "test" {
  id            = %q
  owner_user_id = hindclaw_user.test.id
  display_name  = "TF Test SA"
}`, rUser, rSA),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_service_account.test", "id", rSA),
					resource.TestCheckResourceAttr("hindclaw_service_account.test", "owner_user_id", rUser),
					resource.TestCheckResourceAttr("hindclaw_service_account.test", "display_name", "TF Test SA"),
					resource.TestCheckResourceAttr("hindclaw_service_account.test", "is_active", "true"),
				),
			},
			// Import
			{
				ResourceName:      "hindclaw_service_account.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update display_name
			{
				Config: fmt.Sprintf(`
resource "hindclaw_user" "test" {
  id           = %q
  display_name = "Test User"
}

resource "hindclaw_service_account" "test" {
  id            = %q
  owner_user_id = hindclaw_user.test.id
  display_name  = "TF Test SA Updated"
}`, rUser, rSA),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_service_account.test", "display_name", "TF Test SA Updated"),
				),
			},
		},
	})
}

func TestAccServiceAccountResource_WithScopingPolicy(t *testing.T) {
	rUser := acctest.RandomWithPrefix("tf-test")
	rSA := acctest.RandomWithPrefix("tf-test")
	rPolicy := acctest.RandomWithPrefix("tf-test")

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

resource "hindclaw_policy" "scope" {
  id           = %q
  display_name = "Scoping Policy"
  document     = jsonencode({
    version    = "2026-03-24"
    statements = [{
      effect  = "allow"
      actions = ["bank:recall"]
      banks   = ["yoda"]
    }]
  })
}

resource "hindclaw_service_account" "test" {
  id                = %q
  owner_user_id     = hindclaw_user.test.id
  display_name      = "Scoped SA"
  scoping_policy_id = hindclaw_policy.scope.id
}`, rUser, rPolicy, rSA),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_service_account.test", "scoping_policy_id", rPolicy),
				),
			},
		},
	})
}
