package database

import (
	"context"
	"fmt"
	"maps"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"slices"
)

type Column struct {
	Name             string
	Description      string
	DataType         string
	IsNullable       bool
	IsIndexed        bool
	IsUnique         bool
	IsPrimaryKey     bool
	ForeignKeyTarget string
}

type Table struct {
	Schema      string
	Name        string
	Description string
	Columns     []Column
}

// TODO add in-memory caching
func InspectTables(ctx context.Context, queries *sqlc.Queries, tableNames []string) ([]*Table, error) {
	if len(tableNames) == 0 {
		return nil, fmt.Errorf("at least one table must be specified")
	}

	rows, err := queries.GetTableAndColumnMetadata(ctx, tableNames)

	if err != nil {
		return nil, fmt.Errorf("failed while queryig data: %w", err)
	}

	tableMetadata := map[string]*Table{}
	for _, row := range rows {
		if tableMetadata[row.TableName] == nil {
			tableMetadata[row.TableName] = &Table{
				Name:        row.TableName,
				Description: row.TableDescription,
				Schema:      row.SchemaName,
			}
		}

		table := tableMetadata[row.TableName]

		column := Column{
			Name:             row.ColumnName,
			Description:      row.ColumnDescription,
			DataType:         row.DataType,
			IsNullable:       row.IsNullable,
			IsIndexed:        row.IsIndexed,
			IsUnique:         row.IsUnique,
			IsPrimaryKey:     row.IsPrimaryKey,
			ForeignKeyTarget: row.ForeignKeyTarget,
		}

		table.Columns = append(table.Columns, column)
	}

	return slices.Collect(maps.Values(tableMetadata)), nil
}
