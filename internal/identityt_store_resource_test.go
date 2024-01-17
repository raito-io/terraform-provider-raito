package internal

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccIdentityStoreResource(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			IsUnitTest: false,
			PreCheck: func() {
				AccProviderPreCheck(t)
			},
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.SkipBelow(tfversion.Version1_1_0),
			},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: providerConfig + `
resource "raito_identitystore" "test" {
	name = "tfTestIdentityStore"
	description = "terraform test identity store"
    master = false
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_identitystore.test", "name", "tfTestIdentityStore"),
						resource.TestCheckResourceAttr("raito_identitystore.test", "description", "terraform test identity store"),
						resource.TestCheckResourceAttr("raito_identitystore.test", "master", "false"),
					),
				},
				{
					ResourceName:      "raito_identitystore.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					Config: providerConfig + `
resource "raito_identitystore" "test" {
	name = "tfTestIdentityStore-Rename"
	description = "terraform test identity store renaming description"
    master = true
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_identitystore.test", "name", "tfTestIdentityStore-Rename"),
						resource.TestCheckResourceAttr("raito_identitystore.test", "description", "terraform test identity store renaming description"),
						resource.TestCheckResourceAttr("raito_identitystore.test", "master", "true"),
					),
				},
			},
		})
	})
}
