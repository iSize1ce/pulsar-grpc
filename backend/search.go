package main

import (
	"unicode"

	converter "github.com/alikhil/go-convert-layout"
	"github.com/jmoiron/sqlx"
)

// ruLayout converts between Russian (ЙЦУКЕН) and English (QWERTY) keyboard layouts.
// Used for fuzzy search: when a user types "ghbdtn" (Russian "привет" typed on English layout),
// we also search for the converted variant.
var ruLayout, _ = converter.Create("ru")

// layoutVariant returns the keyboard-layout-converted version of a query string.
// If the query contains Cyrillic, converts to English layout; otherwise converts to Russian.
func layoutVariant(query string) string {
	for _, r := range query {
		if unicode.In(r, unicode.Cyrillic) {
			return ruLayout.ToEn(query)
		}
	}
	return ruLayout.FromEn(query)
}

// searchWithLayoutVariant performs a LIKE search with automatic keyboard layout conversion.
// It tries both the original query and its layout-converted variant so search works
// regardless of which keyboard layout the user has active.
//
//   - simpleQuery:   SQL with 2 LIKE params (original query only)
//   - extendedQuery: SQL with 4 LIKE params (original + layout variant)
func searchWithLayoutVariant[T any](db *sqlx.DB, simpleQuery, extendedQuery, query string) ([]T, error) {
	like := "%" + query + "%"
	alt := layoutVariant(query)

	var items []T
	var err error

	if alt != "" && alt != query {
		likeAlt := "%" + alt + "%"
		err = db.Select(&items, extendedQuery, like, like, likeAlt, likeAlt)
	} else {
		err = db.Select(&items, simpleQuery, like, like)
	}
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = make([]T, 0)
	}
	return items, nil
}
