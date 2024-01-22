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
      user = "user1@company.com"
    },
    {
      user = "user2@company.com",
      promise_duration : 604800
    }
  ]
}