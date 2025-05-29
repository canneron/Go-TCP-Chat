package utils

import "slices"

func Contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
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
