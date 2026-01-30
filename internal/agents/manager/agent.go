package manager

import (
	"context"
	"fmt"
	"strings"

	"github.com/anuvratrastogi/multi-agent/internal/agents/chart"
	sqlagent "github.com/anuvratrastogi/multi-agent/internal/agents/sql"
	"github.com/anuvratrastogi/multi-agent/pkg/bert"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
)

const (
	agentName = "ManagerAgent"
	agentDesc = "Orchestrates SQL queries and data visualization by delegating to specialized sub-agents"
)

// Agent is the Manager agent that routes requests to sub-agents.
type Agent struct {
	agent.Agent
	classifier *bert.Classifier
	sqlAgent   *sqlagent.Agent
	chartAgent *chart.Agent
	llmAgent   agent.Agent
}

// Config holds configuration for the Manager agent.
type Config struct {
	Model      model.LLM
	SQLAgent   *sqlagent.Agent
	ChartAgent *chart.Agent
}

// New creates a new Manager agent with hierarchical sub-agents.
func New(cfg Config) (*Agent, error) {
	classifier := bert.NewClassifier()

	instruction := `You are a manager agent that coordinates between specialized sub-agents.
Your role is to:
1. Understand user requests
2. Route requests to the appropriate sub-agent based on intent
3. Combine results from multiple agents when needed

You have access to two sub-agents:
- SQLAgent: For database queries and SQL operations
- ChartAgent: For data visualization and chart generation

Workflow patterns:
1. SQL-only: User wants data → delegate to SQLAgent
2. Combined: User wants to see data as a chart → first SQLAgent, then ChartAgent with the results

Always provide clear, helpful responses that summarize what was done.`

	llmAgent, err := llmagent.New(llmagent.Config{
		Name:        agentName,
		Description: agentDesc,
		SubAgents:   []agent.Agent{cfg.SQLAgent, cfg.ChartAgent},
		Instruction: instruction,
		Model:       cfg.Model,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create Manager agent: %w", err)
	}

	return &Agent{
		Agent:      llmAgent,
		classifier: classifier,
		sqlAgent:   cfg.SQLAgent,
		chartAgent: cfg.ChartAgent,
		llmAgent:   llmAgent,
	}, nil
}

// ProcessQuery processes a user query by classifying intent and delegating.
func (a *Agent) ProcessQuery(ctx context.Context, query string) (*Result, error) {
	// Classify the intent
	intent, confidence := a.classifier.ClassifyWithConfidence(query)

	result := &Result{
		Query:            query,
		ClassifiedIntent: string(intent),
		Confidence:       confidence,
	}

	// Determine which agents to use based on intent
	switch intent {
	case bert.IntentSQLQuery:
		result.AgentsUsed = []string{"SQLAgent"}
		result.Workflow = "sql_query"

	case bert.IntentVisualization:
		// Visualization might need SQL first if data is mentioned
		if needsDataFetch(query) {
			result.AgentsUsed = []string{"SQLAgent", "ChartAgent"}
			result.Workflow = "sql_then_chart"
		} else {
			result.AgentsUsed = []string{"ChartAgent"}
			result.Workflow = "chart_only"
		}

	default:
		result.AgentsUsed = []string{"ManagerAgent"}
		result.Workflow = "general"
	}

	return result, nil
}

// needsDataFetch determines if the query needs to fetch data first.
func needsDataFetch(query string) bool {
	queryLower := strings.ToLower(query)
	// Simple heuristic: if query mentions data sources, we need SQL first
	dataIndicators := []string{
		"from database", "from table", "data from",
		"show me", "get", "fetch", "query",
		"sales", "users", "orders", "records",
	}

	for _, indicator := range dataIndicators {
		if strings.Contains(queryLower, indicator) {
			return true
		}
	}
	return false
}

// Result represents the result of processing a query.
type Result struct {
	Query            string   `json:"query"`
	ClassifiedIntent string   `json:"classified_intent"`
	Confidence       float64  `json:"confidence"`
	AgentsUsed       []string `json:"agents_used"`
	Workflow         string   `json:"workflow"`
	SQLResult        string   `json:"sql_result,omitempty"`
	ChartResult      string   `json:"chart_result,omitempty"`
	Error            string   `json:"error,omitempty"`
}

// GetClassifier returns the intent classifier.
func (a *Agent) GetClassifier() *bert.Classifier {
	return a.classifier
}

// GetSQLAgent returns the SQL sub-agent.
func (a *Agent) GetSQLAgent() *sqlagent.Agent {
	return a.sqlAgent
}

// GetChartAgent returns the Chart sub-agent.
func (a *Agent) GetChartAgent() *chart.Agent {
	return a.chartAgent
}
