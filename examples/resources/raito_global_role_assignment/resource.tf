resource "raito_user" "u1" {
  name       = "user name"
  email      = "test-user@raito.io"
  raito_user = true
  type       = "Machine"
  password   = "!23vV678"
}

resource "raito_global_role_assignment" "u1_admin" {
  user_id = raito_user.u1.id
  role    = "Admin"
}

resource "raito_global_role_assignment" "u1_creator" {
  user_id = raito_user.u1.id
  role    = "Creator"
}