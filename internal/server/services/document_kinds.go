package services

import "github.com/Volkov-D-A/docs-register-and-track/internal/models"

// DocumentKindCodes returns the codes used in repository filters.
func DocumentKindCodes(kinds []models.DocumentKind) []string {
	codes := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		codes = append(codes, string(kind))
	}
	return codes
}
