package chart

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
)

const (
	agentName      = "ChartAgent"
	agentDesc      = "Creates data visualizations using Mermaid charts from query results"
	outputKeyChart = "chart_result"
)

// Agent is the Chart agent that handles data visualization.
type Agent struct {
	agent.Agent
}

// Config holds configuration for the Chart agent.
type Config struct {
	Model model.LLM
}

// New creates a new Chart agent.
func New(cfg Config) (*Agent, error) {
	instruction := `You are a data visualization expert agent. Your job is to:
1. Analyze the data provided (usually from SQL query results)
2. Determine the most appropriate chart type for the data
3. Generate a Mermaid chart in markdown format

Mermaid Chart Types Available:
- xychart-beta: For bar charts and line charts (use for comparisons and trends)
- pie: For showing proportions of a whole

Output Format:
Return your response with the chart in a mermaid code block. Use this format:

For bar/line charts:
` + "```mermaid" + `
xychart-beta
    title "Chart Title"
    x-axis [Label1, Label2, Label3]
    y-axis "Y Axis Label" MIN --> MAX
    bar [value1, value2, value3]
` + "```" + `

For line charts:
` + "```mermaid" + `
xychart-beta
    title "Chart Title"
    x-axis [Label1, Label2, Label3]  
    y-axis "Y Axis Label" MIN --> MAX
    line [value1, value2, value3]
` + "```" + `

For pie charts:
` + "```mermaid" + `
pie title "Chart Title"
    "Label1" : value1
    "Label2" : value2
    "Label3" : value3
` + "```" + `

IMPORTANT Guidelines:
- Set y-axis MIN to 0 and MAX to slightly above your highest data value (e.g., if max value is 135, use 0 --> 150)
- Choose chart type based on data characteristics
- Use clear, descriptive titles and labels
- For time series data, prefer line charts (xychart-beta with line)
- For category comparisons, prefer bar charts (xychart-beta with bar)
- For proportions of a whole, prefer pie charts
- Keep labels short to fit in the chart
- Round numbers appropriately for readability
- Always output valid Mermaid syntax

Generate clean, readable Mermaid charts that can be rendered in any markdown viewer.`

	llmAgent, err := llmagent.New(llmagent.Config{
		Name:        agentName,
		Description: agentDesc,
		Instruction: instruction,
		Model:       cfg.Model,
		OutputKey:   outputKeyChart,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create Chart agent: %w", err)
	}

	return &Agent{Agent: llmAgent}, nil
}

// ChartConfig represents the configuration for a chart.
type ChartConfig struct {
	ChartType string       `json:"chart_type"`
	Title     string       `json:"title"`
	Data      ChartData    `json:"data"`
	Options   ChartOptions `json:"options"`
	Mermaid   string       `json:"mermaid,omitempty"`
}

// ChartData represents the data for a chart.
type ChartData struct {
	Labels   []string  `json:"labels"`
	Datasets []Dataset `json:"datasets"`
}

// Dataset represents a single dataset in a chart.
type Dataset struct {
	Label string    `json:"label"`
	Data  []float64 `json:"data"`
}

// ChartOptions represents chart options.
type ChartOptions struct {
	XAxisLabel string `json:"x_axis_label,omitempty"`
	YAxisLabel string `json:"y_axis_label,omitempty"`
}

// ParseChartConfig parses the agent's output into a ChartConfig.
func ParseChartConfig(output string) (*ChartConfig, error) {
	var config ChartConfig
	if err := json.Unmarshal([]byte(output), &config); err != nil {
		return nil, fmt.Errorf("failed to parse chart config: %w", err)
	}
	return &config, nil
}

// GenerateMermaidBarChart generates a Mermaid bar chart from data.
func GenerateMermaidBarChart(title string, labels []string, data []float64, yAxisLabel string) string {
	maxVal := 0.0
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}
	// Round up to nice number
	maxVal = roundUpNice(maxVal)

	labelsStr := "[" + strings.Join(quoteLabels(labels), ", ") + "]"
	dataStr := "[" + joinFloats(data) + "]"

	return fmt.Sprintf("```mermaid\nxychart-beta\n    title \"%s\"\n    x-axis %s\n    y-axis \"%s\" 0 --> %.0f\n    bar %s\n```",
		title, labelsStr, yAxisLabel, maxVal, dataStr)
}

// GenerateMermaidLineChart generates a Mermaid line chart from data.
func GenerateMermaidLineChart(title string, labels []string, data []float64, yAxisLabel string) string {
	maxVal := 0.0
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}
	maxVal = roundUpNice(maxVal)

	labelsStr := "[" + strings.Join(quoteLabels(labels), ", ") + "]"
	dataStr := "[" + joinFloats(data) + "]"

	return fmt.Sprintf("```mermaid\nxychart-beta\n    title \"%s\"\n    x-axis %s\n    y-axis \"%s\" 0 --> %.0f\n    line %s\n```",
		title, labelsStr, yAxisLabel, maxVal, dataStr)
}

// GenerateMermaidPieChart generates a Mermaid pie chart from data.
func GenerateMermaidPieChart(title string, labels []string, data []float64) string {
	var parts []string
	for i, label := range labels {
		if i < len(data) {
			parts = append(parts, fmt.Sprintf("    \"%s\" : %.0f", label, data[i]))
		}
	}

	return fmt.Sprintf("```mermaid\npie title \"%s\"\n%s\n```",
		title, strings.Join(parts, "\n"))
}

func quoteLabels(labels []string) []string {
	quoted := make([]string, len(labels))
	for i, l := range labels {
		quoted[i] = fmt.Sprintf("\"%s\"", l)
	}
	return quoted
}

func joinFloats(data []float64) string {
	strs := make([]string, len(data))
	for i, v := range data {
		strs[i] = fmt.Sprintf("%.0f", v)
	}
	return strings.Join(strs, ", ")
}

func roundUpNice(val float64) float64 {
	if val <= 10 {
		return 10
	} else if val <= 50 {
		return 50
	} else if val <= 100 {
		return 100
	} else if val <= 500 {
		return 500
	} else if val <= 1000 {
		return 1000
	}
	// Round up to nearest 1000
	return float64(int(val/1000)+1) * 1000
}

func toJSONString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
