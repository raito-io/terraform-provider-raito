variable "raito_password" {
  description = "Raito password"
  type        = string
  sensitive   = true
}


provider "raito" {
  domain       = "app"
  user         = "ruben+cli@raito.io"
  secret       = var.raito_password
  url_override = "https://api.raito.dev"
}

resource "raito_datasource" "tfDs1" {
  name        = "tfDs1"
  description = "testing terraform datasource"
}

resource "raito_grant" "grant1" {
  name = "TerraformGrant1"
  description = "Terraform grant 1"
  data_source = "lR0QWfLtP7TsZcphexSvx"
  what_data_objects = [
    {
      name = "THOMAS_TEST.TESTING.CITIES"
      global_permissions = ["READ", "WRITE"]
    }
  ]
  state = "Inactive"
  who = [
    {
      user = "a-yasirellis5644@raito.io"
    },
    {
      user = "a.ralphross7988@raito.io"
    },

  ]
}