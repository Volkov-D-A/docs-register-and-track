package ports

type DocumentCommandPrincipal interface {
	DocumentAccessPrincipal
	RequireSystemPermission(string) error
}
