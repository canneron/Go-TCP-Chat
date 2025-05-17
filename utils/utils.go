package utils

func Contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func RemoveValue[T comparable](slice []T, value T) []T {
	result := []T{}
	for _, v := range slice {
		if v != value {
			result = append(result, v)
		}
	}
	return result
}
