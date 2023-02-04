package server

import "math"

const (
	nbFilmsPerPage int64 = 20
)

type Pagination struct {
	Number int64
	Active bool
	Dots   bool
}

// getPagination creates a Pagination slice
func getPagination[T any](currentPage int64, items []T) ([]T, []Pagination) {
	var pages []Pagination
	pageMax := int64(math.Ceil(float64(len(items)) / float64(nbFilmsPerPage)))

	pages = append(pages, Pagination{
		Number: 1,
		Active: currentPage == 1,
	})
	// Add dots to link between 1 and current-1
	if currentPage > 3 {
		pages = append(pages, Pagination{
			Dots: true,
		})
	}
	for i := currentPage - 1; i <= currentPage+1; i++ {
		if i <= 1 || i >= pageMax {
			continue
		}
		if i == currentPage {
			pages = append(pages, Pagination{
				Number: i,
				Active: true,
			})
		} else {
			pages = append(pages, Pagination{
				Number: i,
			})
		}
	}
	// Add dots to link between current+1 and max
	if currentPage < pageMax-2 {
		pages = append(pages, Pagination{
			Dots: true,
		})
	}
	if pageMax > 1 {
		pages = append(pages, Pagination{
			Number: pageMax,
			Active: currentPage == pageMax,
		})
	}

	// Return only part of the items (corresponding to the current page)
	itemsIndexStart := (currentPage - 1) * nbFilmsPerPage
	itemsIndexEnd := itemsIndexStart + nbFilmsPerPage

	var pagedItems []T
	for i := itemsIndexStart; i < itemsIndexEnd && i < int64(len(items)); i++ {
		pagedItems = append(pagedItems, items[i])
	}

	return pagedItems, pages
}
