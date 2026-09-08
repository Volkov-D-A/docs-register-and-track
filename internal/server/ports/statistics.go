package ports

type StatisticsPrincipal interface {
	RequireAuthenticated() error
	HasSystemPermission(string) bool
}
