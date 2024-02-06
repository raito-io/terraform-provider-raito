package internal

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccPurposeResource(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			IsUnitTest: false,
			PreCheck: func() {
				AccProviderPreCheck(t)
			},
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.SkipBelow(tfversion.Version1_0_0),
			},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: providerConfig + `
data "raito_datasource" "ds" {
    name = "Snowflake"
}

resource "raito_purpose" "purpose1" {
	name = "tfPurpose1"
	description = "purpose description"
	state = "Inactive"
	who = [
        {
            user             = "terraform@raito.io"
		}
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_purpose.purpose1", "name", "tfPurpose1"),
						resource.TestCheckResourceAttr("raito_purpose.purpose1", "description", "purpose description"),
						resource.TestCheckResourceAttr("raito_purpose.purpose1", "who.#", "1"),
						resource.TestCheckResourceAttr("raito_purpose.purpose1", "who.0.user", "terraform@raito.io"),
						resource.TestCheckNoResourceAttr("raito_purpose.purpose1", "what.0.promise_duration"),
					),
				},
				{
					ResourceName:            "raito_purpose.purpose1",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "what"},
				},
				{
					Config: providerConfig + `
data "raito_datasource" "ds" {
    name = "Snowflake"
}

resource "raito_purpose" "purpose1" {
	name = "tfPurpose1-update"
	description = "updated terraform purpose"
	state = "Active"
	who = [
        {
            user             = "terraform@raito.io"
			promise_duration = 604800
		}
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_purpose.purpose1", "name", "tfPurpose1-update"),
						resource.TestCheckResourceAttr("raito_purpose.purpose1", "description", "updated terraform purpose"),
						resource.TestCheckResourceAttr("raito_purpose.purpose1", "who.#", "1"),
						resource.TestCheckResourceAttr("raito_purpose.purpose1", "who.0.user", "terraform@raito.io"),
						resource.TestCheckResourceAttr("raito_purpose.purpose1", "who.0.promise_duration", "604800"),
					),
				},
				{
					Config: providerConfig + `
data "raito_datasource" "ds" {
    name = "Snowflake"
}

locals {
	abac_rule = jsonencode({
		aggregator: {
			operator: "Or",
			operands: [
				{
					aggregator: {
						operator: "And",
						operands: [
							{
								comparison: {
									operator: "HasTag"
									leftOperand: "Test"
									rightOperand: {
										literal: { string: "test" }
									}
								}
							}
						]
					}
				}
			]
		}
	})
}

resource "raito_purpose" "purpose1" {
	name = "tfPurpose1-update"
	description = "updated terraform purpose"
	state = "Active"
	who_abac_rule = local.abac_rule
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_purpose.purpose1", "name", "tfPurpose1-update"),
						resource.TestCheckResourceAttr("raito_purpose.purpose1", "description", "updated terraform purpose"),
						resource.TestCheckNoResourceAttr("raito_purpose.purpose1", "who"),
						resource.TestCheckResourceAttr("raito_purpose.purpose1", "who_abac_rule", "{\"aggregator\":{\"operands\":[{\"aggregator\":{\"operands\":[{\"comparison\":{\"leftOperand\":\"Test\",\"operator\":\"HasTag\",\"rightOperand\":{\"literal\":{\"string\":\"test\"}}}}],\"operator\":\"And\"}}],\"operator\":\"Or\"}}"),
					),
				},
			},
		})
	})
}
