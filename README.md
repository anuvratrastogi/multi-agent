# Multi-Agent System with Google ADK for Go

A hierarchical multi-agent system using Google's Agent Development Kit (ADK) for Go that processes natural language queries to interact with PostgreSQL databases and generate data visualizations.

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      Manager Agent                            │
│              (BERT-based Intent Classification)               │
├──────────────────────┬───────────────────────────────────────┤
│                      │                                        │
▼                      ▼                                        │
┌──────────────────────────────────┐  ┌────────────────────────┐
│           SQL Agent              │  │      Chart Agent       │
│   (Text-to-SQL via MCP Tools)    │  │  (Data Visualization)  │
└───────────────┬──────────────────┘  └────────────────────────┘
                │
                ▼
┌──────────────────────────────────┐
│     PostgreSQL MCP Server        │
│  (query, schema, list_tables)    │
└──────────────────────────────────┘
```

## Features

- **Manager Agent**: Uses BERT-style intent classification to route queries to specialized agents
- **SQL Agent**: Converts natural language to SQL queries using Gemini LLM and MCP tools
- **Chart Agent**: Generates interactive charts (bar, line, pie, scatter) using Chart.js
- **MCP PostgreSQL Server**: Exposes database tools for schema introspection and query execution

## Prerequisites

- Go 1.24+
- PostgreSQL database
- Google Cloud API key (for Gemini)

## Installation

```bash
cd /home/anuvrat/code/github.com/anuvratrastogi/multi-agent
go mod download
go build -o multi-agent ./cmd/main.go
```

## Configuration

Set the following environment variables:

### Option 1: Using Gemini (Default)

```bash
export DATABASE_URL="postgres://user:password@localhost:5432/dbname?sslmode=disable"
export GOOGLE_API_KEY="your-gemini-api-key"
export GEMINI_MODEL="gemini-2.0-flash"  # Optional
```

### Option 2: Using Local LLM (e.g., LM Studio)

```bash
export DATABASE_URL="postgres://user:password@localhost:5432/dbname?sslmode=disable"
export LLM_PROVIDER="local"
export LOCAL_LLM_URL="http://localhost:1234"
export LLM_MODEL="local-model" # Optional
```

Ensure your local LLM server (like LM Studio) is running and accessible at the specified URL.

## Usage

```bash
./multi-agent
```

### Example Queries

```
You: Show me all tables in the database
📋 Intent: sql_query (confidence: 0.75)
🔄 Workflow: sql_query
🤖 Agents: SQLAgent

You: Create a bar chart of sales by month
📋 Intent: visualization (confidence: 0.82)
🔄 Workflow: sql_then_chart
🤖 Agents: SQLAgent → ChartAgent

You: How many users are in the database?
📋 Intent: sql_query (confidence: 0.68)
🔄 Workflow: sql_query
🤖 Agents: SQLAgent
```

## Project Structure

```
multi-agent/
├── cmd/
│   └── main.go                 # Entry point with REPL interface
├── config/
│   └── config.go               # Environment configuration
├── internal/
│   ├── agents/
│   │   ├── manager/
│   │   │   └── agent.go        # Manager agent with intent routing
│   │   ├── sql/
│   │   │   ├── agent.go        # SQL agent with MCP tools
│   │   │   └── client.go       # Direct PostgreSQL client
│   │   └── chart/
│   │       └── agent.go        # Chart generation agent
│   └── mcp/
│       └── server.go           # PostgreSQL MCP server
└── pkg/
    └── bert/
        └── classifier.go       # Intent classification
```

## MCP Tools

The MCP PostgreSQL server exposes the following tools:

| Tool | Description |
|------|-------------|
| `query_database` | Execute SQL queries and return JSON results |
| `get_schema` | Get table schema (columns, types, constraints) |
| `list_tables` | List all tables in public schema |
| `describe_database` | Get complete database structure overview |

## Intent Classification

The classifier recognizes three intent types:

- **sql_query**: Queries about data retrieval, database structure
- **visualization**: Requests for charts, graphs, visualizations
- **general**: Help, explanations, general questions

## Technologies

- **[Google ADK for Go](https://github.com/google/adk-go)**: Agent Development Kit
- **[MCP Go](https://github.com/mark3labs/mcp-go)**: Model Context Protocol implementation
- **[Chart.js](https://www.chartjs.org/)**: Chart rendering (embedded in HTML output)
- **[Gemini](https://ai.google.dev/)**: LLM for text-to-SQL and chart configuration

## License

MIT
