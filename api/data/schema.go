package data

import (
	"strings"

	"github.com/atombasedev/atombase/tools"
)

// // parseDefaultValue converts SQLite's default value string to appropriate Go type
// func parseDefaultValue(val string) any {
// 	// Remove quotes from string defaults
// 	if len(val) >= 2 && ((val[0] == '\'' && val[len(val)-1] == '\'') || (val[0] == '"' && val[len(val)-1] == '"')) {
// 		return val[1 : len(val)-1]
// 	}
// 	// Try to parse as number
// 	if val == "NULL" || val == "null" {
// 		return nil
// 	}
// 	// Return as-is for expressions like CURRENT_TIMESTAMP
// 	return val
// }

// SearchFks searches for a foreign key from table to references.
// Returns the Fk and true if found, or empty Fk and false if not found.
func (schema SchemaCache) SearchFks(table string, references string) (CacheFk, bool) {
	fks, exists := schema.Fks[table]
	if !exists {
		return CacheFk{}, false
	}
	for _, fk := range fks {
		if fk.References == references {
			return fk, true
		}
	}
	return CacheFk{}, false
}

// SearchTbls searches for a table by name.
// Returns the Table or ErrTableNotFound if not found.
func (schema SchemaCache) SearchTbls(table string) (CacheTable, error) {
	tbl, exists := schema.Tables[table]
	if !exists {
		return CacheTable{}, tools.TableNotFoundErr(table)
	}
	return tbl, nil
}

// SearchCols searches a column by name.
// Returns the Col or ErrColumnNotFound if not found.
func (tbl CacheTable) SearchCols(col string) (string, error) {
	c, exists := tbl.Columns[col]
	if !exists {
		return "", tools.ColumnNotFoundErr(tbl.Name, col)
	}
	return c, nil
}

// HasFTSIndex checks if a table has an FTS5 index.
func (schema SchemaCache) HasFTSIndex(table string) bool {
	return schema.FTSTables[table]
}

// BuildColumnTypeMap builds a flat map of column name -> type from all tables.
// Types are normalized to uppercase (TEXT, INTEGER, REAL, BLOB) for consistent matching.
// For columns with the same name in different tables, the type is taken from the first table found.
func (schema SchemaCache) BuildColumnTypeMap() map[string]string {
	result := make(map[string]string)
	for _, table := range schema.Tables {
		for colName, colType := range table.Columns {
			if _, exists := result[colName]; !exists {
				result[colName] = strings.ToUpper(colType)
			}
		}
	}
	return result
}
