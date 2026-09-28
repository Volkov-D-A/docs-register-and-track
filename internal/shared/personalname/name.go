// Package personalname defines the shared rules for user name components.
package personalname

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const MaxComponentLength = 100

func Normalize(lastName, firstName, patronymic string, noPatronymic bool) (string, string, string, error) {
	lastName, firstName, patronymic = strings.TrimSpace(lastName), strings.TrimSpace(firstName), strings.TrimSpace(patronymic)
	if noPatronymic {
		patronymic = ""
	}
	for _, field := range []struct{ label, value string }{{"Фамилия", lastName}, {"Имя", firstName}, {"Отчество", patronymic}} {
		if field.value == "" && (field.label != "Отчество" || !noPatronymic) {
			return "", "", "", fmt.Errorf("%s: обязательное поле", field.label)
		}
		if utf8.RuneCountInString(field.value) > MaxComponentLength {
			return "", "", "", fmt.Errorf("%s: максимум %d символов", field.label, MaxComponentLength)
		}
	}
	return lastName, firstName, patronymic, nil
}

func Display(lastName, firstName, patronymic string, noPatronymic bool) string {
	parts := []string{}
	for i, value := range []string{lastName, firstName, patronymic} {
		if i == 2 && noPatronymic {
			break
		}
		if value = strings.TrimSpace(value); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " ")
}
