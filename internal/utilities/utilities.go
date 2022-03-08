package utilities

import "strings"

// RemoveArticle removes an article from the beginning of a string
func RemoveArticle(input string) string {
	if strings.HasPrefix(strings.ToLower(input), "a ") {
		return input[2:]
	} else if strings.HasPrefix(strings.ToLower(input), "an ") {
		return input[3:]
	} else if strings.HasPrefix(strings.ToLower(input), "the ") {
		return input[4:]
	}
	return input
}

// Int64SliceContains returns true if the slice contains the element
// TODO: use generics when available
func Int64SliceContains(slice []int64, elt int64) bool {
	for _, a := range slice {
		if a == elt {
			return true
		}
	}
	return false
}
