package main

import (
	"strings"
)

// TableRef contains data for referenced tables
type TableRef struct {
	noCopy  noCopy
	Table   *Table     // name of the table itself, ex.: hosts table
	Columns ColumnList // local column(s) which holds the values to determince the ID of the referenced item, ex.: host_name
}

// Table defines available columns and table options
type Table struct {
	noCopy          noCopy
	Name            string
	Columns         ColumnList
	ColumnsIndex    map[string]*Column // access columns by name
	PassthroughOnly bool               // flag wether table will be cached or simply passed through to remote sites
	GroupBy         bool
	PrimaryKey      []string
	RefTables       []*TableRef // referenced tables
	Virtual         VirtStoreResolveFunc
	DefaultSort     []string // columns used to sort if nothing is specified
}

// GetColumn returns a column for given name or nil if not found
func (t *Table) GetColumn(name string) *Column {
	return t.ColumnsIndex[name]
}

// GetColumnWithFallback returns a column for list of names
func (t *Table) GetColumnWithFallback(name string) *Column {
	col, ok := t.ColumnsIndex[name]
	if ok {
		return col
	}
	if !fixBrokenClientsRequestColumn(&name, t.Name) {
		return t.GetEmptyColumn()
	}
	return t.ColumnsIndex[name]
}

// GetColumns returns a column list for list of names
func (t *Table) GetColumns(names []string) *ColumnList {
	columns := make(ColumnList, 0, len(names))
	for i := range names {
		columns = append(columns, t.ColumnsIndex[names[i]])
	}
	return &columns
}

// GetEmptyColumn returns an empty column
func (t *Table) GetEmptyColumn() *Column {
	return &Column{
		Name:        "empty",
		Description: "placeholder for unknown columns",
		Table:       t,
		DataType:    StringCol,
		StorageType: VirtStore,
		FetchType:   None,
		VirtMap:     VirtColumnMap["empty"],
	}
}

// AddColumn adds a new column
func (t *Table) AddColumn(name string, update FetchType, datatype DataType, description string) {
	NewColumn(t, name, LocalStore, update, datatype, NoFlags, nil, description)
}

// AddExtraColumn adds a new column with extra attributes
func (t *Table) AddExtraColumn(name string, storage StorageType, update FetchType, datatype DataType, restrict OptionalFlags, description string) {
	NewColumn(t, name, storage, update, datatype, restrict, nil, description)
}

// AddPeerStatusColumns adds a new column related to peer information
func (t *Table) AddPeerInfoColumn(name string, datatype DataType, description string) {
	NewColumn(t, name, VirtStore, None, datatype, NoFlags, nil, description)
}

// AddRefColumns adds a reference column.
// tableName: name of the referenced table
// Prefix: column prefix for the added columns
// LocalName: column(s) which holds the reference value(s)
func (t *Table) AddRefColumns(tableName string, prefix string, localName []string) {
	refTable, Ok := Objects.Tables[tableName]
	if !Ok {
		log.Panicf("no such reference %s from column %s", tableName, strings.Join(localName, ","))
	}

	t.RefTables = append(t.RefTables, &TableRef{Table: refTable, Columns: *(t.GetColumns(localName))})

	// add fake columns for all columns from the referenced table
	for i := range Objects.Tables[tableName].Columns {
		col := Objects.Tables[tableName].Columns[i]
		// skip peer_key and such things from ref table
		if col.StorageType == RefStore {
			continue
		}
		refColName := prefix + "_" + col.Name
		if prefix == "" {
			refColName = col.Name
		}
		if _, ok := t.ColumnsIndex[refColName]; ok {
			continue
		}
		NewColumn(t, refColName, RefStore, None, col.DataType, col.Optional, col, col.Description)
	}
}

// SetColumnIndex sets index for all columns
func (t *Table) SetColumnIndex() {
	flagCombos := []OptionalFlags{
		NoFlags,
		Shinken,
		Icinga2,
		Naemon | Naemon1_0_10,
	}
	for _, flags := range flagCombos {
		indexes := make(map[DataType]int)
		for i := range t.Columns {
			col := t.Columns[i]
			if col.Optional != NoFlags && !flags.HasFlag(col.Optional) {
				continue
			}
			if col.StorageType == LocalStore {
				_, ok := indexes[col.DataType]
				if !ok {
					indexes[col.DataType] = 0
				}
				if col.Index != indexes[col.DataType] && col.Index > 0 {
					// overlapping indexes would break data storage, make sure that columns for flags that include
					// other flags come last, ex.: first set columns for flag Naemon, then add colums for Naemon1_10
					log.Panicf("index overlap with flags in column %s of table %s: %v / %d != %d", col.Name, t.Name, flags.String(), col.Index, indexes[col.DataType])
				}
				col.Index = indexes[col.DataType]
				indexes[col.DataType]++
			}
		}
	}
}
