package business

import (
	"math"

	"github.com/Agurato/starfin/internal/model"
)

// PaginaterWrapper implements the GetPagination method
type PaginaterWrapper[T any] struct {
	itemsPerPage int64
}

// NewPaginaterWrapper instantiates a new PaginaterWrapper
func NewPaginaterWrapper[T any](itemsPerPage int64) *PaginaterWrapper[T] {
	return &PaginaterWrapper[T]{
		itemsPerPage: itemsPerPage,
	}
}

// GetPagination creates a slice of the input elements and Pagination slice
func (pw *PaginaterWrapper[T]) GetPagination(currentPage int64, items []T) ([]T, []model.Pagination) {
	var pages []model.Pagination
	pageMax := int64(math.Ceil(float64(len(items)) / float64(pw.itemsPerPage)))

	pages = append(pages, model.Pagination{
		Number: 1,
		Active: currentPage == 1,
	})
	// Add dots to link between 1 and current-1
	if currentPage > 3 {
		pages = append(pages, model.Pagination{
			Dots: true,
		})
	}
	for i := currentPage - 1; i <= currentPage+1; i++ {
		if i <= 1 || i >= pageMax {
			continue
		}
		if i == currentPage {
			pages = append(pages, model.Pagination{
				Number: i,
				Active: true,
			})
		} else {
			pages = append(pages, model.Pagination{
				Number: i,
			})
		}
	}
	// Add dots to link between current+1 and max
	if currentPage < pageMax-2 {
		pages = append(pages, model.Pagination{
			Dots: true,
		})
	}
	if pageMax > 1 {
		pages = append(pages, model.Pagination{
			Number: pageMax,
			Active: currentPage == pageMax,
		})
	}

	// Return only part of the items (corresponding to the current page)
	itemsIndexStart := (currentPage - 1) * pw.itemsPerPage
	itemsIndexEnd := itemsIndexStart + pw.itemsPerPage

	var pagedItems []T
	for i := itemsIndexStart; i < itemsIndexEnd && i < int64(len(items)); i++ {
		pagedItems = append(pagedItems, items[i])
	}

	return pagedItems, pages
}
