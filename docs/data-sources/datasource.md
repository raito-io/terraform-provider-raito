---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "raito_datasource Data Source - terraform-provider-raito"
subcategory: ""
description: |-
  Find datasource based on the name
---

# raito_datasource (Data Source)

Find datasource based on the name

## Example Usage

```terraform
data "raito_datasource" "example" {
  name = "Snowflake"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the data source to

### Read-Only

- `description` (String) Description of the data source
- `id` (String) ID of the requested data source
- `identity_stores` (Set of String) Linked identity stores
- `native_identity_store` (String) ID of the native identity store
- `parent` (String) Parent data source id if applicable
- `sync_method` (String) Sync method of the data source