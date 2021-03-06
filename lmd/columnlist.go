package main

import (
	"strings"
)

// ColumnList defines a set of Columns
type ColumnList []*Column

// String returns the string representation of a column list
func (dl *ColumnList) String() string {
	names := make([]string, 0, len(*dl))
	for i := range *dl {
		names = append(names, (*dl)[i].String())
	}
	return strings.Join(names, ", ")
}
