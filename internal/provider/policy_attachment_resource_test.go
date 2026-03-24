package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPolicyAttachmentResource_Group(t *testing.T) {
	rPolicy := acctest.RandomWithPrefix("tf-test")
	rGroup := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "hindclaw_group" "test" {
  id           = %q
  display_name = "Test Group"
}

resource "hindclaw_policy" "test" {
  id           = %q
  display_name = "Test Policy"
  document     = jsonencode({
    version    = "2026-03-24"
    statements = [{
      effect  = "allow"
      actions = ["bank:recall"]
      banks   = ["*"]
    }]
  })
}

resource "hindclaw_policy_attachment" "test" {
  policy_id      = hindclaw_policy.test.id
  principal_type = "group"
  principal_id   = hindclaw_group.test.id
  priority       = 0
}`, rGroup, rPolicy),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_policy_attachment.test", "principal_type", "group"),
					resource.TestCheckResourceAttr("hindclaw_policy_attachment.test", "priority", "0"),
				),
			},
			// Import
			{
				ResourceName:                         "hindclaw_policy_attachment.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "policy_id",
				ImportStateId:                        fmt.Sprintf("%s/group/%s", rPolicy, rGroup),
			},
			// Update priority
			{
				Config: fmt.Sprintf(`
resource "hindclaw_group" "test" {
  id           = %q
  display_name = "Test Group"
}

resource "hindclaw_policy" "test" {
  id           = %q
  display_name = "Test Policy"
  document     = jsonencode({
    version    = "2026-03-24"
    statements = [{
      effect  = "allow"
      actions = ["bank:recall"]
      banks   = ["*"]
    }]
  })
}

resource "hindclaw_policy_attachment" "test" {
  policy_id      = hindclaw_policy.test.id
  principal_type = "group"
  principal_id   = hindclaw_group.test.id
  priority       = 10
}`, rGroup, rPolicy),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_policy_attachment.test", "priority", "10"),
				),
			},
		},
	})
}

func TestAccPolicyAttachmentResource_User(t *testing.T) {
	rPolicy := acctest.RandomWithPrefix("tf-test")
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

resource "hindclaw_policy" "test" {
  id           = %q
  display_name = "Test Policy"
  document     = jsonencode({
    version    = "2026-03-24"
    statements = [{
      effect  = "allow"
      actions = ["bank:recall"]
      banks   = ["*"]
    }]
  })
}

resource "hindclaw_policy_attachment" "test" {
  policy_id      = hindclaw_policy.test.id
  principal_type = "user"
  principal_id   = hindclaw_user.test.id
}`, rUser, rPolicy),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_policy_attachment.test", "principal_type", "user"),
					resource.TestCheckResourceAttr("hindclaw_policy_attachment.test", "priority", "0"),
				),
			},
		},
	})
}
