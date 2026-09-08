package services

import "github.com/google/uuid"

func UUIDStrings(ids []uuid.UUID) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != uuid.Nil {
			result = append(result, id.String())
		}
	}
	return result
}

func AppendUniqueUserID(items []uuid.UUID, item uuid.UUID) []uuid.UUID {
	if item == uuid.Nil {
		return items
	}
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}
