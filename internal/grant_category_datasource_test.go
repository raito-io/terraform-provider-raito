package internal

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccGrantCategoryDataSource(t *testing.T) {
	t.Run("Existing purpose", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			IsUnitTest: false,
			PreCheck: func() {
				AccProviderPreCheck(t)
			},
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.SkipBelow(tfversion.Version1_0_0),
			},
			Steps: []resource.TestStep{
				{
					Config: providerConfig + `
data "raito_grant_category" "test" {
	name = "Purpose"
}
					`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("data.raito_grant_category.test", "name", "Purpose"),
						resource.TestCheckResourceAttr("data.raito_grant_category.test", "is_system", "true"),
					),
				},
			},
		})
	})
}
