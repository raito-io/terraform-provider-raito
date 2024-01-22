package types

const (
	GlobalPermissionRead  = "READ"
	GlobalPermissionWrite = "WRITE"
	GlobalPermissionAdmin = "ADMIN"
)

var AllGlobalPermissions = []string{
	GlobalPermissionRead,
	GlobalPermissionWrite,
	GlobalPermissionAdmin,
}
