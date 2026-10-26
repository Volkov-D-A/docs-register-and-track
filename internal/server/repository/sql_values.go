package repository

// nullableSQLValue maps optional string values to SQL NULL without changing
// non-empty values; SQL casts define whether the value is a UUID, date or text.
func nullableSQLValue(value string) any {
	if value == "" {
		return nil
	}
	return value
}
