package services

import (
	"fmt"

	"github.com/google/uuid"

	"docflow/internal/repository"
)

// parseUUID — парсинг строки UUID с обработкой ошибок
func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID: %s", s)
	}
	return id, nil
}

// formatDocumentNumber — форматирование номера документа
func formatDocumentNumber(index string, number int) string {
	return fmt.Sprintf("%s/%d", index, number)
}

// filterNomenclaturesByDepartment — общая логика фильтрации номенклатур по подразделению пользователя.
// Возвращает отфильтрованный список nomenclatureIDs, флаг isEmpty (если нужно вернуть пустой результат),
// и ошибку.
// Параметры:
//   - departmentID: ID подразделения пользователя (nil — нет подразделения → пустой результат)
//   - depRepo: репозиторий подразделений для получения разрешённых номенклатур
//   - filterNomIDs: уже заданные в фильтре NomenclatureIDs (множественный фильтр)
//   - filterNomID: одиночный NomenclatureID из фильтра (используется, если filterNomIDs пуст)
func filterNomenclaturesByDepartment(
	departmentID *uuid.UUID,
	depRepo *repository.DepartmentRepository,
	filterNomIDs []string,
	filterNomID string,
) (filteredIDs []string, isEmpty bool, err error) {
	if departmentID == nil {
		// Исполнитель без отдела ничего не видит
		return nil, true, nil
	}

	allowedNomenclatures, err := depRepo.GetNomenclatureIDs(*departmentID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get allowed nomenclatures: %w", err)
	}

	if len(allowedNomenclatures) == 0 {
		return nil, true, nil
	}

	// Если фильтр по номенклатурам уже задан — вычисляем пересечение
	if len(filterNomIDs) > 0 {
		allowedMap := make(map[string]bool, len(allowedNomenclatures))
		for _, id := range allowedNomenclatures {
			allowedMap[id] = true
		}
		var intersection []string
		for _, id := range filterNomIDs {
			if allowedMap[id] {
				intersection = append(intersection, id)
			}
		}
		if len(intersection) == 0 {
			return nil, true, nil
		}
		return intersection, false, nil
	}

	// Если задан одиночный ID — проверяем его наличие в разрешённых
	if filterNomID != "" {
		for _, id := range allowedNomenclatures {
			if id == filterNomID {
				return nil, false, nil // ID остаётся в фильтре как есть
			}
		}
		// Недоступна → пустой результат
		return nil, true, nil
	}

	// Фильтр пустой → подставляем все доступные номенклатуры
	return allowedNomenclatures, false, nil
}
