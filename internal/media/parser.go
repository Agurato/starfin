package media

import (
	"github.com/dlclark/regexp2"
)

func ParseFilename(filename string) (string, string, error) {
	var (
		movieTitle string
		year       string
	)
	movieTitleRegex := regexp2.MustCompile(`^(?<title>(?![(\[]).+?)?(?:(?:[-_\W](?<![)\[!]))*(?<year>(1(8|9)|20)\d{2}(?!p|i|(1(8|9)|20)\d{2}|\]|\W(1(8|9)|20)\d{2})))+(\W+|_|$)(?!\\)`, regexp2.IgnoreCase)

	if ok, err := movieTitleRegex.MatchString(filename); ok {
		matches, err := movieTitleRegex.FindStringMatch(filename)
		if err != nil {
			return "", "", err
		}
		movieTitle = matches.GroupByName("title").String()
		year = matches.GroupByName("year").String()
	} else {
		return "", "", err
	}

	return movieTitle, year, nil
}
