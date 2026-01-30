package sql

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

const (
	agentName    = "SQLAgent"
	agentDesc    = "Converts natural language queries to SQL and executes them against the PostgreSQL database"
	outputKeySQL = "sql_result"
)

// Agent is the SQL agent that handles text-to-SQL conversion.
type Agent struct {
	agent.Agent
}

// Config holds configuration for the SQL agent.
type Config struct {
	Model          model.LLM
	Tools          []tool.Tool
	DatabaseSchema string // Optional: pre-loaded database schema for better SQL generation
}

// New creates a new SQL agent.
func New(cfg Config) (*Agent, error) {
	instruction := `You are a SQL expert agent. Your job is to:
1. Understand the user's natural language query about data
2. Convert it to a valid PostgreSQL query
3. Execute the query using the available database tools
4. Return the results in a structured format

Guidelines:
- Write efficient SQL queries with appropriate WHERE clauses
- Limit results to a reasonable number unless specifically asked for all
- Format dates and numbers appropriately
- If the query is ambiguous, make reasonable assumptions and explain them
- Use the database schema provided below to write accurate queries

Available tools:
- query_database: Execute SQL queries and get results
- get_schema: Get the schema of a specific table (if you need more details)
- list_tables: List all available tables
- describe_database: Get an overview of the database structure`

	// Add database schema to instruction if provided
	if cfg.DatabaseSchema != "" {
		instruction += "\n\n## Database Schema\n" + cfg.DatabaseSchema
	}

	instruction += "\n\nAlways return the query results as structured JSON data."

	llmAgent, err := llmagent.New(llmagent.Config{
		Name:        agentName,
		Description: agentDesc,
		Instruction: instruction,
		Model:       cfg.Model,
		Tools:       cfg.Tools,
		OutputKey:   outputKeySQL,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create SQL agent: %w", err)
	}

	return &Agent{Agent: llmAgent}, nil
}

// QueryResult represents the result of a SQL query.
type QueryResult struct {
	Query   string                   `json:"query"`
	Data    []map[string]interface{} `json:"data"`
	Count   int                      `json:"count"`
	Columns []string                 `json:"columns,omitempty"`
	Error   string                   `json:"error,omitempty"`
}

// ParseResult parses the agent's output into a QueryResult.
func ParseResult(output string) (*QueryResult, error) {
	var result QueryResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		// If it's not a structured result, wrap it
		return &QueryResult{
			Data:  nil,
			Error: "",
		}, nil
	}
	return &result, nil
}

// Tool argument and result types for functiontool
type QueryArgs struct {
	SQL   string `json:"sql" description:"The SQL query to execute"`
	Limit int    `json:"limit,omitempty" description:"Maximum number of rows to return (default: 100)"`
}

type QueryResult2 struct {
	Data  string `json:"data"`
	Error string `json:"error,omitempty"`
}

type SchemaArgs struct {
	TableName string `json:"table_name" description:"The name of the table to get schema for"`
}

type SchemaResult struct {
	Schema string `json:"schema"`
	Error  string `json:"error,omitempty"`
}

type EmptyArgs struct{}

type ListTablesResult struct {
	Tables string `json:"tables"`
	Error  string `json:"error,omitempty"`
}

type DescribeResult struct {
	Description string `json:"description"`
	Error       string `json:"error,omitempty"`
}

// CreateMCPTools creates the MCP tools for the SQL agent using functiontool.
func CreateMCPTools(mcpClient MCPClient) ([]tool.Tool, error) {
	var tools []tool.Tool

	// Query database tool
	queryTool, err := functiontool.New(
		functiontool.Config{
			Name:        "query_database",
			Description: "Execute a SQL query and return results as JSON",
		},
		func(ctx tool.Context, args QueryArgs) (QueryResult2, error) {
			fmt.Printf("  📝 [SQL] Executing query: %s\n", args.SQL)
			limit := args.Limit
			if limit == 0 {
				limit = 100
			}
			data, err := mcpClient.Query(context.Background(), args.SQL, limit)
			if err != nil {
				fmt.Printf("  ❌ [SQL] Query error: %v\n", err)
				return QueryResult2{Error: err.Error()}, nil
			}
			fmt.Printf("  ✅ [SQL] Query completed successfully\n")
			return QueryResult2{Data: data}, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create query_database tool: %w", err)
	}
	tools = append(tools, queryTool)

	// Get schema tool
	schemaTool, err := functiontool.New(
		functiontool.Config{
			Name:        "get_schema",
			Description: "Get the schema of a specific table",
		},
		func(ctx tool.Context, args SchemaArgs) (SchemaResult, error) {
			fmt.Printf("  📋 [TOOL] get_schema: %s\n", args.TableName)
			schema, err := mcpClient.GetSchema(context.Background(), args.TableName)
			if err != nil {
				return SchemaResult{Error: err.Error()}, nil
			}
			return SchemaResult{Schema: schema}, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create get_schema tool: %w", err)
	}
	tools = append(tools, schemaTool)

	// List tables tool
	listTablesTool, err := functiontool.New(
		functiontool.Config{
			Name:        "list_tables",
			Description: "List all tables in the database",
		},
		func(ctx tool.Context, args EmptyArgs) (ListTablesResult, error) {
			fmt.Printf("  📋 [TOOL] list_tables\n")
			tables, err := mcpClient.ListTables(context.Background())
			if err != nil {
				return ListTablesResult{Error: err.Error()}, nil
			}
			return ListTablesResult{Tables: tables}, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create list_tables tool: %w", err)
	}
	tools = append(tools, listTablesTool)

	// Describe database tool
	describeTool, err := functiontool.New(
		functiontool.Config{
			Name:        "describe_database",
			Description: "Get an overview of the database structure including all tables and their columns",
		},
		func(ctx tool.Context, args EmptyArgs) (DescribeResult, error) {
			fmt.Printf("  📋 [TOOL] describe_database\n")
			desc, err := mcpClient.DescribeDatabase(context.Background())
			if err != nil {
				return DescribeResult{Error: err.Error()}, nil
			}
			return DescribeResult{Description: desc}, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create describe_database tool: %w", err)
	}
	tools = append(tools, describeTool)

	return tools, nil
}

// MCPClient interface for database operations.
type MCPClient interface {
	Query(ctx context.Context, query string, limit int) (string, error)
	GetSchema(ctx context.Context, tableName string) (string, error)
	ListTables(ctx context.Context) (string, error)
	DescribeDatabase(ctx context.Context) (string, error)
}
