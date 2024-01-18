resource "raito_datasource" "ds" {
  name = "exampleDS"
}

resource "raito_grant" "grant1" {
  name        = "First grant"
  description = "A simple grant"
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
  type        = "role"
  data_source = raito_datasource.ds.id
  what_data_objects = {
    data_object : [
      {
        name : "data_object1"
        permissions : ["SELECT", "INSERT"]
        global_permissions : []
      },
      {
        name : "data_object2"
        global_permissions : ["READ"]
      }
    ]
  }
}

resource "raito_grant" "grant2" {
  name        = "Grant2"
  description = "Grant with inherited who"
  state       = "Active"
  who = [
    {
      access_control = raito_grant.grant1.id
    }
  ]
  data_source = raito_datasource.ds.id
}