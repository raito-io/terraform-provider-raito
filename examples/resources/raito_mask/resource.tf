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