---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "raito_mask Resource - terraform-provider-raito"
subcategory: ""
description: |-
  Mask access control resource
---

# raito_mask (Resource)

Mask access control resource

## Example Usage

```terraform
resource "raito_datasource" "ds" {
  name = "exampleDS"
}

resource "raito_mask" "example" {
  name        = "A Raito Mask"
  description = "A simple mask"
  state       = "Active"
  who = [
    {
      user : "ruben@raito.io"
    },
    {
      user : "dieter@raito.io"
      promise_duration : 604800
    }
  ]
  type        = "SHA256"
  data_source = raito_datasource.ds.id
  columns = [
    "SOME_DB.SOME_SCHEMA.SOME_TABLE.SOME_COLUMN"
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `data_source` (String) Data source ID of the mask
- `name` (String) Name of the mask

### Optional

- `columns` (Set of String) Full name of columns that should be included in the mask. Items are managed by Raito Cloud if columns is not set (nil).
- `description` (String) Description of the mask
- `state` (String) State of the mask. Possible values: ["Active", "Inactive"]
- `type` (String) Type of the mask. This defines how the data is masked. Available types are defined by the data source.
- `who` (Attributes Set) Who items associated to the mask. May not be set if who_abac_rule is set. Items are managed by Raito Cloud of who is not set (nil) (see [below for nested schema](#nestedatt--who))

### Read-Only

- `id` (String) ID of the mask

<a id="nestedatt--who"></a>
### Nested Schema for `who`

Optional:

- `access_control` (String) Raito access control ID. Can not be set if user or group is set.
- `group` (String) Raito group ID. Can not be set if user or access control is set.
- `promise_duration` (Number) Set promise_duration to indicate who item as promise. Defined in seconds.
- `user` (String) Email address of user. Can not be set if group or access control is set.

## Import

Import is supported using the following syntax:

```shell
terraform import raito_mask.example MaskId
```