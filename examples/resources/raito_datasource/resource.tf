resource "raito_datasource" "example" {
  name        = "DataSourceName"
  description = "A description for the data source"
  sync_method = "ON_PREM"
  parent      = "ParentId"
  IdentityStores = [
    "linked_identity_store_id"
  ]
}