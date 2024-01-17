package internal

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	raitoType "github.com/raito-io/sdk/types"
)

func TestAccDataSourceResource(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			IsUnitTest: false,
			PreCheck: func() {
				AccProviderPreCheck(t)
			},
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.SkipBelow(tfversion.Version1_1_0),
			},
			Steps: []resource.TestStep{
				{
					Config: providerConfig + `
resource "raito_datasource" "test" {
	name        = "tfTestDataSource"
	description = "test description"
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_datasource.test", "name", "tfTestDataSource"),
						resource.TestCheckResourceAttr("raito_datasource.test", "description", "test description"),
						resource.TestCheckResourceAttr("raito_datasource.test", "sync_method", string(raitoType.DataSourceSyncMethodOnPrem)),
						resource.TestCheckResourceAttr("raito_datasource.test", "identity_stores.#", "0"),
						resource.TestCheckNoResourceAttr("raito_datasource.test", "parent"),
						resource.TestCheckResourceAttrWith("raito_datasource.test", "native_identity_store", func(value string) error {
							if value == "" {
								return errors.New("native_identity_store should not be empty")
							}

							return nil
						}),
					),
				},
				{
					ResourceName:      "raito_datasource.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					Config: providerConfig + `
resource "raito_datasource" "test" {
	name        = "tfTestDataSourceUpdateName"
	description = "test update description"
	sync_method = "CLOUD_MANUAL_TRIGGER"
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_datasource.test", "name", "tfTestDataSourceUpdateName"),
						resource.TestCheckResourceAttr("raito_datasource.test", "description", "test update description"),
						resource.TestCheckResourceAttr("raito_datasource.test", "sync_method", string(raitoType.DataSourceSyncMethodCloudManualTrigger)),
					),
				},
				// Resource are automatically deleted
			},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		})
	})

	t.Run("link_identity_stores", func(t *testing.T) {
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
resource "raito_identitystore" "tfIs1" {
	name = "tfIs1DataSourceTest"
}

resource "raito_identitystore" "tfIs2" {
    name = "tfIs2DataSourceTest"
}

resource "raito_datasource" "test" {
	name = "tfDs1"
	identity_stores = [
		raito_identitystore.tfIs1.id,
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_datasource.test", "name", "tfDs1"),
						resource.TestCheckResourceAttr("raito_datasource.test", "identity_stores.#", "1"),
						resource.TestCheckResourceAttrPair("raito_identitystore.tfIs1", "id", "raito_datasource.test", "identity_stores.0"),
					),
				},
				{
					ResourceName:      "raito_datasource.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
				{
					Config: providerConfig + `
resource "raito_identitystore" "tfIs1" {
	name = "tfIs1DataSourceTest"
}

resource "raito_identitystore" "tfIs2" {
    name = "tfIs2DataSourceTest"
}

resource "raito_datasource" "test" {
	name = "tfDs1"
	identity_stores = [
		raito_identitystore.tfIs2.id,
	]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("raito_datasource.test", "name", "tfDs1"),
						resource.TestCheckResourceAttr("raito_datasource.test", "identity_stores.#", "1"),
						resource.TestCheckResourceAttrPair("raito_identitystore.tfIs2", "id", "raito_datasource.test", "identity_stores.0"),
					),
				},
				{
					ResourceName:      "raito_datasource.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})
}
