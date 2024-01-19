---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "raito_purpose Resource - terraform-provider-raito"
subcategory: ""
description: |-
  Purpose access control resource
---

# raito_purpose (Resource)

Purpose access control resource

## Example Usage

```terraform
resource "raito_datasource" "ds" {
  name = "exampleDS"
}

resource "raito_grant" "grant1" {
  name = "An existing grant"
}

resource "raito_purpose" "example_purpose" {
  name        = "Example Purpose"
  description = "This is an example purpose"
  state       = "Inactive"
  what = [
    raito_grant.grant1.id
  ]
  who = [
    {
      user = "ruben@raito.io"
    },
    {
      user = "dieter@raito.io",
      promise_duration : 604800
    }
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the purpose

### Optional

- `description` (String) Description of the purpose
- `state` (String) State of the purpose. Possible values: ["Active", "Inactive"]
- `type` (String) Type of the purpose
- `what` (Set of String) What items associated tot the purpose. items are managed by Raito Cloud if what is not set (nil).
- `who` (Attributes Set) Who items associated to the purpose. May not be set if who_abac_rule is set. Items are managed by Raito Cloud of who is not set (nil) (see [below for nested schema](#nestedatt--who))

### Read-Only

- `id` (String) ID of the purpose

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
terraform import raito_purpose.example PurposeId
```