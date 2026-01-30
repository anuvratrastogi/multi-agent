package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

// DirectMCPClient is a direct database client implementing MCPClient interface.
type DirectMCPClient struct {
	db *sql.DB
}

// NewDirectMCPClient creates a new direct MCP client.
func NewDirectMCPClient(databaseURL string) (*DirectMCPClient, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DirectMCPClient{db: db}, nil
}

// Query executes a SQL query and returns results as JSON.
func (c *DirectMCPClient) Query(ctx context.Context, query string, limit int) (string, error) {
	// Add LIMIT if not present and it's a SELECT query
	queryUpper := strings.ToUpper(strings.TrimSpace(query))
	if strings.HasPrefix(queryUpper, "SELECT") && !strings.Contains(queryUpper, "LIMIT") {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return "", fmt.Errorf("scan error: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for JSON serialization
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	jsonResult, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("json error: %w", err)
	}

	return string(jsonResult), nil
}

// GetSchema returns the schema of a table as JSON.
func (c *DirectMCPClient) GetSchema(ctx context.Context, tableName string) (string, error) {
	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := c.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return "", fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var schema []map[string]interface{}
	for rows.Next() {
		var columnName, dataType, isNullable string
		var columnDefault sql.NullString

		if err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault); err != nil {
			return "", fmt.Errorf("scan error: %w", err)
		}

		col := map[string]interface{}{
			"column_name": columnName,
			"data_type":   dataType,
			"nullable":    isNullable == "YES",
		}
		if columnDefault.Valid {
			col["default"] = columnDefault.String
		}
		schema = append(schema, col)
	}

	jsonResult, err := json.Marshal(schema)
	if err != nil {
		return "", fmt.Errorf("json error: %w", err)
	}

	return string(jsonResult), nil
}

// ListTables returns a list of tables as JSON.
func (c *DirectMCPClient) ListTables(ctx context.Context) (string, error) {
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		ORDER BY table_name
	`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return "", fmt.Errorf("scan error: %w", err)
		}
		tables = append(tables, tableName)
	}

	jsonResult, err := json.Marshal(tables)
	if err != nil {
		return "", fmt.Errorf("json error: %w", err)
	}

	return string(jsonResult), nil
}

// DescribeDatabase returns database structure as JSON.
func (c *DirectMCPClient) DescribeDatabase(ctx context.Context) (string, error) {
	query := `
		SELECT 
			t.table_name,
			array_agg(c.column_name || ' ' || c.data_type ORDER BY c.ordinal_position) as columns
		FROM information_schema.tables t
		JOIN information_schema.columns c ON t.table_name = c.table_name AND t.table_schema = c.table_schema
		WHERE t.table_schema = 'public'
		GROUP BY t.table_name
		ORDER BY t.table_name
	`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var tables []map[string]interface{}
	for rows.Next() {
		var tableName string
		var columns []string

		if err := rows.Scan(&tableName, &columns); err != nil {
			return "", fmt.Errorf("scan error: %w", err)
		}

		tables = append(tables, map[string]interface{}{
			"table":   tableName,
			"columns": columns,
		})
	}

	jsonResult, err := json.Marshal(tables)
	if err != nil {
		return "", fmt.Errorf("json error: %w", err)
	}

	return string(jsonResult), nil
}

// Close closes the database connection.
func (c *DirectMCPClient) Close() error {
	return c.db.Close()
}
