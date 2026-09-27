package dto

import "github.com/Volkov-D-A/docs-register-and-track/internal/models"

func MapUser(m *models.User) *User {
	if m == nil {
		return nil
	}
	var department *Department
	if m.Department != nil {
		department = MapDepartment(m.Department)
	}
	return &User{ID: m.ID.String(), Login: m.Login, FullName: m.FullName, IsDocumentParticipant: m.IsDocumentParticipant, IsActive: m.IsActive, FailedLoginAttempts: m.FailedLoginAttempts, SystemPermissions: m.SystemPermissions, Department: department}
}
func MapUserSubstitution(m *models.UserSubstitution) *UserSubstitution {
	if m == nil {
		return nil
	}
	return &UserSubstitution{SubstituteUserID: m.SubstituteUserID.String(), StartsAt: m.StartsAt, EndsAt: m.EndsAt, IsActive: m.IsActive}
}
func MapDepartment(m *models.Department) *Department {
	if m == nil {
		return nil
	}
	return &Department{ID: m.ID.String(), Name: m.Name, NomenclatureIDs: m.NomenclatureIDs}
}
func MapNomenclature(m *models.Nomenclature) *Nomenclature {
	if m == nil {
		return nil
	}
	return &Nomenclature{ID: m.ID.String(), Name: m.Name, Index: m.Index, Year: m.Year, KindCode: m.KindCode, Separator: m.Separator, NumberingMode: m.NumberingMode, NextNumber: m.NextNumber, IsActive: m.IsActive}
}
func MapOrganization(m *models.Organization) *Organization {
	if m == nil {
		return nil
	}
	return &Organization{ID: m.ID.String(), Name: m.Name}
}
func MapUsers(m []models.User) []User {
	if m == nil {
		return nil
	}
	result := make([]User, len(m))
	for i, item := range m {
		if mapped := MapUser(&item); mapped != nil {
			result[i] = *mapped
		}
	}
	return result
}
func MapDepartments(m []models.Department) []Department {
	if m == nil {
		return nil
	}
	result := make([]Department, len(m))
	for i, item := range m {
		if mapped := MapDepartment(&item); mapped != nil {
			result[i] = *mapped
		}
	}
	return result
}
func MapNomenclatures(m []models.Nomenclature) []Nomenclature {
	if m == nil {
		return nil
	}
	result := make([]Nomenclature, len(m))
	for i, item := range m {
		if mapped := MapNomenclature(&item); mapped != nil {
			result[i] = *mapped
		}
	}
	return result
}
func MapOrganizations(m []models.Organization) []Organization {
	if m == nil {
		return nil
	}
	result := make([]Organization, len(m))
	for i, item := range m {
		if mapped := MapOrganization(&item); mapped != nil {
			result[i] = *mapped
		}
	}
	return result
}
