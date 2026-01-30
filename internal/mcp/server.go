package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// PostgresServer wraps the MCP server for PostgreSQL operations.
type PostgresServer struct {
	server *server.MCPServer
	db     *sql.DB
}

// NewPostgresServer creates a new PostgreSQL MCP server.
func NewPostgresServer(databaseURL string) (*PostgresServer, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := server.NewMCPServer(
		"PostgreSQL MCP Server",
		"1.0.0",
	)

	ps := &PostgresServer{
		server: s,
		db:     db,
	}

	ps.registerTools()

	return ps, nil
}

// registerTools registers all PostgreSQL tools with the MCP server.
func (ps *PostgresServer) registerTools() {
	// Query database tool
	queryTool := mcp.NewTool("query_database",
		mcp.WithDescription("Execute a SQL query and return results as JSON"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The SQL query to execute"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of rows to return (default: 100)"),
		),
	)

	ps.server.AddTool(queryTool, ps.handleQuery)

	// Get schema tool
	schemaTool := mcp.NewTool("get_schema",
		mcp.WithDescription("Get the schema of a specific table"),
		mcp.WithString("table_name",
			mcp.Required(),
			mcp.Description("The name of the table to get schema for"),
		),
	)

	ps.server.AddTool(schemaTool, ps.handleGetSchema)

	// List tables tool
	listTablesTool := mcp.NewTool("list_tables",
		mcp.WithDescription("List all tables in the database"),
	)

	ps.server.AddTool(listTablesTool, ps.handleListTables)

	// Describe database tool
	describeTool := mcp.NewTool("describe_database",
		mcp.WithDescription("Get an overview of the database structure including all tables and their columns"),
	)

	ps.server.AddTool(describeTool, ps.handleDescribeDatabase)
}

// handleQuery executes a SQL query and returns results.
func (ps *PostgresServer) handleQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	query, ok := args["query"].(string)
	if !ok {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	limit := 100.0
	if l, ok := args["limit"].(float64); ok {
		limit = l
	}

	// Add LIMIT if not present and it's a SELECT query
	queryUpper := strings.ToUpper(strings.TrimSpace(query))
	if strings.HasPrefix(queryUpper, "SELECT") && !strings.Contains(queryUpper, "LIMIT") {
		// Strip trailing semicolon if present
		query = strings.TrimSpace(query)
		if strings.HasSuffix(query, ";") {
			query = query[:len(query)-1]
		}
		query = fmt.Sprintf("%s LIMIT %d", query, int(limit))
	}

	rows, err := ps.db.QueryContext(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("query error: %v", err)), nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get columns: %v", err)), nil
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("scan error: %v", err)), nil
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	jsonResult, err := json.Marshal(results)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("json error: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

// handleGetSchema returns the schema of a table.
func (ps *PostgresServer) handleGetSchema(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	tableName, ok := args["table_name"].(string)
	if !ok {
		return mcp.NewToolResultError("table_name parameter is required"), nil
	}

	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := ps.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("query error: %v", err)), nil
	}
	defer rows.Close()

	var schema []map[string]interface{}
	for rows.Next() {
		var columnName, dataType, isNullable string
		var columnDefault sql.NullString

		if err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("scan error: %v", err)), nil
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
		return mcp.NewToolResultError(fmt.Sprintf("json error: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

// handleListTables lists all tables in the database.
func (ps *PostgresServer) handleListTables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		ORDER BY table_name
	`

	rows, err := ps.db.QueryContext(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("query error: %v", err)), nil
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("scan error: %v", err)), nil
		}
		tables = append(tables, tableName)
	}

	jsonResult, err := json.Marshal(tables)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("json error: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

// handleDescribeDatabase provides an overview of the database.
func (ps *PostgresServer) handleDescribeDatabase(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := `
		SELECT 
			t.table_name,
			array_agg(c.column_name || ' ' || c.data_type ORDER BY c.ordinal_position) as columns
		FROM information_schema.tables t
		JOIN information_schema.columns c ON t.table_name = c.table_name
		WHERE t.table_schema = 'public'
		GROUP BY t.table_name
		ORDER BY t.table_name
	`

	rows, err := ps.db.QueryContext(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("query error: %v", err)), nil
	}
	defer rows.Close()

	var tables []map[string]interface{}
	for rows.Next() {
		var tableName string
		var columns []string

		if err := rows.Scan(&tableName, pq.Array(&columns)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("scan error: %v", err)), nil
		}

		tables = append(tables, map[string]interface{}{
			"table":   tableName,
			"columns": columns,
		})
	}

	jsonResult, err := json.Marshal(tables)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("json error: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

// GetServer returns the underlying MCP server.
func (ps *PostgresServer) GetServer() *server.MCPServer {
	return ps.server
}

// Close closes the database connection.
func (ps *PostgresServer) Close() error {
	return ps.db.Close()
}

// ServeStdio starts the server using stdio transport.
func (ps *PostgresServer) ServeStdio() error {
	return server.ServeStdio(ps.server)
}
