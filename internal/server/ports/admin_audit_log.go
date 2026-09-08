package ports

type SystemPermissionPrincipal interface {
	RequireSystemPermission(string) error
}
