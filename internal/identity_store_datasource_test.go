package internal

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccIdentityStoreDataSource(t *testing.T) {
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
				Config: providerConfig + `data "raito_identity_store" "test" {
				name = "Snowflake"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.raito_identity_store.test", "name", "Snowflake"),
					resource.TestCheckResourceAttrWith("data.raito_identity_store.test", "id", func(value string) error {
						if value == "" {
							return errors.New("ID is not set")
						}

						return nil
					}),
					resource.TestCheckNoResourceAttr("data.raito_identity_store.test", "owners.0"),
					resource.TestCheckResourceAttr("data.raito_identity_store.test", "master", "false"),
					resource.TestCheckResourceAttr("data.raito_identity_store.test", "is_native", "true"),
				),
			},
		},
	})
}
