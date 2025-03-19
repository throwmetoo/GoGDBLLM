package utils

// Contains checks if a slice contains a specific element
func Contains[T comparable](slice []T, element T) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}

// Filter returns a new slice containing only the elements for which the predicate returns true
func Filter[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Map applies a function to each element in a slice and returns a new slice with the results
func Map[T any, U any](slice []T, mapper func(T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = mapper(item)
	}
	return result
}

// Unique returns a new slice with duplicate elements removed
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{}, len(slice))
	result := make([]T, 0, len(slice))

	for _, item := range slice {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

// Intersection returns a slice containing elements that are in both slices
func Intersection[T comparable](a, b []T) []T {
	set := make(map[T]struct{})
	for _, item := range a {
		set[item] = struct{}{}
	}

	result := make([]T, 0)
	for _, item := range b {
		if _, ok := set[item]; ok {
			result = append(result, item)
		}
	}

	return result
}
