variable "raito_password" {
  description = "Raito password"
  type        = string
  sensitive   = true
}

variable "raito_user_password" {
  description = "Raito user password"
  type        = string
  sensitive   = true
}

provider "raito" {
  domain       = "app"
  user         = "ruben+cli@raito.io"
  secret       = var.raito_password
  url_override = "https://api.raito.dev"
}

#resource "raito_identitystore" "tfIS1" {
#  name        = "tfIS1"
#  description = "Identity store created by terraform"
#}
#
#resource "raito_datasource" "tfDS1" {
#  name        = "tfDS1"
#  description = "Data source created by terraform"
#  identity_stores = [
#    raito_identitystore.tfIS1.id,
#  ]
#}

data "raito_datasource" "snowflakeDS" {
  name = "Snowflake"
}

resource "raito_purpose" "purpose1" {
  name        = "Terraform purpose"
  description = "testing a purpose"
  who = [
    {
      user = "a-yasirellis5644@raito.io"
    },
    {
      user             = "ruben@raito.io"
      promise_duration = 604800
    }
  ]
}

resource "raito_grant" "grant1" {
  name        = "TerraformGrant1"
  description = "Terraform grant 1 - test123"
  data_source = data.raito_datasource.snowflakeDS.id
  what_data_objects = [
    {
      fullname            = "THOMAS_TEST.TESTING.CITIES"
      global_permissions = ["READ", "WRITE"]
    }
  ]
  who = [
    {
      access_control = raito_purpose.purpose1.id
    }
  ]
}

#resource "raito_mask" "mask1" {
#  name        = "tfMask1"
#  description = "Testing to create a mask with terraform"
#  data_source = data.raito_datasource.snowflakeDS.id
#  type        = "SHA256"
#  columns = [
#    "RUBEN_TEST.TESTING.CITIES.CITY"
#  ]
#  who = [
#    {
#      user             = "a-yasirellis5644@raito.io"
#      promise_duration = 604800
#    },
#    {
#      access_control = raito_purpose.purpose1.id
#    }
#  ]
#}
#
#resource "raito_user" "u1" {
#  name = "TF user"
#  email = "ruben+terraform@raito.io"
#  raito_user = true
#  type = "Human"
#  password = var.raito_user_password
#}
#