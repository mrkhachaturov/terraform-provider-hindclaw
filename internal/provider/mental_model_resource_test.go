package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMentalModelResource_trigger(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with trigger
			{
				Config: `
resource "hindclaw_mental_model" "test" {
  bank_id      = "yoda"
  name         = "TF Trigger Test"
  source_query = "What trigger behavior is configured?"

  trigger = {
    refresh_after_consolidation = true
    fact_types                  = ["experience", "observation"]
    exclude_mental_models       = true
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hindclaw_mental_model.test", "id"),
					resource.TestCheckResourceAttr("hindclaw_mental_model.test", "trigger.refresh_after_consolidation", "true"),
					resource.TestCheckResourceAttr("hindclaw_mental_model.test", "trigger.fact_types.#", "2"),
					resource.TestCheckTypeSetElemAttr("hindclaw_mental_model.test", "trigger.fact_types.*", "experience"),
					resource.TestCheckTypeSetElemAttr("hindclaw_mental_model.test", "trigger.fact_types.*", "observation"),
					resource.TestCheckResourceAttr("hindclaw_mental_model.test", "trigger.exclude_mental_models", "true"),
				),
			},
			// Step 2: Import — ignore optional list fields inside trigger
			// that may round-trip as null vs empty after import (no prior
			// state for stringSliceToTFListPreserveNullOnEmpty).
			{
				ResourceName:            "hindclaw_mental_model.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateIdFunc:       importMentalModelStateIdFunc("hindclaw_mental_model.test"),
				ImportStateVerifyIgnore: []string{"trigger.fact_types", "trigger.exclude_mental_model_ids"},
			},
			// Step 3: Update trigger — change all fields to verify update path
			{
				Config: `
resource "hindclaw_mental_model" "test" {
  bank_id      = "yoda"
  name         = "TF Trigger Test"
  source_query = "What trigger behavior is configured?"

  trigger = {
    refresh_after_consolidation = false
    fact_types                  = ["world"]
    exclude_mental_models       = false
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hindclaw_mental_model.test", "trigger.refresh_after_consolidation", "false"),
					resource.TestCheckResourceAttr("hindclaw_mental_model.test", "trigger.fact_types.#", "1"),
					resource.TestCheckTypeSetElemAttr("hindclaw_mental_model.test", "trigger.fact_types.*", "world"),
					resource.TestCheckResourceAttr("hindclaw_mental_model.test", "trigger.exclude_mental_models", "false"),
				),
			},
		},
	})
}

func TestAccMentalModelResource_triggerOmitted(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without trigger — server defaults applied
			{
				Config: `
resource "hindclaw_mental_model" "test_no_trigger" {
  bank_id      = "yoda"
  name         = "TF No Trigger Test"
  source_query = "What is the architecture?"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hindclaw_mental_model.test_no_trigger", "id"),
					// Server always returns a trigger with defaults
					resource.TestCheckResourceAttr("hindclaw_mental_model.test_no_trigger", "trigger.refresh_after_consolidation", "false"),
					resource.TestCheckResourceAttr("hindclaw_mental_model.test_no_trigger", "trigger.exclude_mental_models", "false"),
				),
			},
		},
	})
}

// importMentalModelStateIdFunc builds the {bank_id}/{id} import ID from state.
func importMentalModelStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("not found: %s", resourceName)
		}
		return rs.Primary.Attributes["bank_id"] + "/" + rs.Primary.Attributes["id"], nil
	}
}
