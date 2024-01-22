package internal

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccGrantResource(t *testing.T) {
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

resource "raito_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_source = data.raito_datasource.ds.id
	what_data_objects = [
		{
			"fullname": "MASTER_DATA.SALES"
		}
	]
	who = [
		{
			"user": "terraform@raito.io"
		}
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("raito_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("raito_grant.test", "data_source", "data.raito_datasource.ds", "id"),
						resource.TestCheckResourceAttr("raito_grant.test", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("raito_grant.test", "what_data_objects.0.fullname", "MASTER_DATA.SALES"),
						resource.TestCheckResourceAttr("raito_grant.test", "who.#", "1"),
						resource.TestCheckResourceAttr("raito_grant.test", "who.0.user", "terraform@raito.io"),
					),
				},
				{
					ResourceName:            "raito_grant.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"who", "what_data_objects"},
				},
				{
					Config: providerConfig + fmt.Sprintf(`
data "raito_datasource" "ds" {
    name = "Snowflake"
}

resource "raito_grant" "test" {
	name        = "tfTestGrant"
    description = "test description"
	data_source = data.raito_datasource.ds.id
	state = "Inactive"
	what_data_objects = [
		{
			fullname: "MASTER_DATA.SALES"
			permissions: ["SELECT"]
		}
	]
	who = [
		{
			"user": "terraform@raito.io"
		}
	]
}
`),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_grant.test", "name", "tfTestGrant"),
						resource.TestCheckResourceAttr("raito_grant.test", "description", "test description"),
						resource.TestCheckResourceAttrPair("raito_grant.test", "data_source", "data.raito_datasource.ds", "id"),
						resource.TestCheckResourceAttr("raito_grant.test", "what_data_objects.#", "1"),
						resource.TestCheckResourceAttr("raito_grant.test", "what_data_objects.0.fullname", "MASTER_DATA.SALES"),
						resource.TestCheckResourceAttr("raito_grant.test", "what_data_objects.0.permissions.#", "1"),
						resource.TestCheckResourceAttr("raito_grant.test", "what_data_objects.0.permissions.0", "SELECT"),
						resource.TestCheckResourceAttr("raito_grant.test", "who.#", "1"),
						resource.TestCheckResourceAttr("raito_grant.test", "who.0.user", "terraform@raito.io"),
					),
				},
			},
		})
	})
}
