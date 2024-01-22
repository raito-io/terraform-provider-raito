package internal

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccMaskResource(t *testing.T) {
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

resource "raito_mask" "test" {
	name        = "tfTestMask"
    description = "test description"
	data_source = data.raito_datasource.ds.id
	columns = []
	who = [
     {
       user             = "terraform@raito.io"
       promise_duration = 604800
     }
   ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_mask.test", "name", "tfTestMask"),
						resource.TestCheckResourceAttr("raito_mask.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("raito_mask.test", "data_source", "data.raito_datasource.ds", "id"),
						resource.TestCheckResourceAttr("raito_mask.test", "columns.#", "0"),
						resource.TestCheckResourceAttr("raito_mask.test", "who.#", "1"),
						resource.TestCheckResourceAttr("raito_mask.test", "who.0.user", "terraform@raito.io"),
						resource.TestCheckResourceAttr("raito_mask.test", "who.0.promise_duration", "604800"),
						resource.TestCheckResourceAttr("raito_mask.test", "type", "NULL"),
					),
				},
				{
					ResourceName:            "raito_mask.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "columns"},
				},
				{
					Config: providerConfig + `data "raito_datasource" "ds" {
    name = "Snowflake"
}

resource "raito_mask" "test" {
	name        = "Terraform Mask name edit"
    description = "test description"
	data_source = data.raito_datasource.ds.id
	who = [
     {
       user             = "terraform@raito.io"
     }
   ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_mask.test", "name", "Terraform Mask name edit"),
						resource.TestCheckResourceAttr("raito_mask.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("raito_mask.test", "data_source", "data.raito_datasource.ds", "id"),
						resource.TestCheckNoResourceAttr("raito_mask.test", "columns"),
						resource.TestCheckResourceAttr("raito_mask.test", "who.#", "1"),
						resource.TestCheckResourceAttr("raito_mask.test", "who.0.user", "terraform@raito.io"),
						resource.TestCheckNoResourceAttr("raito_mask.test", "who.0.promise_duration"),
						resource.TestCheckResourceAttr("raito_mask.test", "type", "NULL"),
					),
				},
			},
		})
	})
}
