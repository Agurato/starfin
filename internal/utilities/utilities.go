package utilities

import "strings"

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
