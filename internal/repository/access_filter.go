package repository

func accessibleUserIDs(primary string, additional []string) []string {
	seen := make(map[string]struct{}, len(additional)+1)
	result := make([]string, 0, len(additional)+1)
	if primary != "" {
		seen[primary] = struct{}{}
		result = append(result, primary)
	}
	for _, id := range additional {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
