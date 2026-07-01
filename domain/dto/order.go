package dto

import "gorm.io/gorm/clause"

type Direction string

const (
	DESC Direction = "DESC"
	ASC  Direction = "ASC"
)

// OrderByCol builds a single-column ORDER BY clause.
func OrderByCol(column string, direction Direction) *clause.OrderBy {
	return &clause.OrderBy{
		Columns: []clause.OrderByColumn{
			{Column: clause.Column{Name: column}, Desc: direction == DESC},
		},
	}
}

// OrderByDesc builds a multi-column ORDER BY clause, all descending.
func OrderByDesc(columns ...string) *clause.OrderBy {
	cols := make([]clause.OrderByColumn, len(columns))
	for i, col := range columns {
		cols[i] = clause.OrderByColumn{Column: clause.Column{Name: col}, Desc: true}
	}
	return &clause.OrderBy{Columns: cols}
}
