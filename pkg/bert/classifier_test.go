package bert

import (
	"testing"
)

func TestClassifier_Classify(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name     string
		query    string
		expected Intent
	}{
		{
			name:     "SQL query - show tables",
			query:    "Show me all tables in the database",
			expected: IntentSQLQuery,
		},
		{
			name:     "SQL query - get users",
			query:    "Get all users from the users table",
			expected: IntentSQLQuery,
		},
		{
			name:     "SQL query - select data",
			query:    "Select all records where status is active",
			expected: IntentSQLQuery,
		},
		{
			name:     "Visualization - bar chart",
			query:    "Create a bar chart of sales by month",
			expected: IntentVisualization,
		},
		{
			name:     "Visualization - line graph",
			query:    "Show me a line graph of revenue over time",
			expected: IntentVisualization,
		},
		{
			name:     "Visualization - pie chart",
			query:    "Generate a pie chart showing market share",
			expected: IntentVisualization,
		},
		{
			name:     "General - help",
			query:    "How do I use this system?",
			expected: IntentGeneral,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Classify(tt.query)
			if got != tt.expected {
				t.Errorf("Classify(%q) = %v, want %v", tt.query, got, tt.expected)
			}
		})
	}
}

func TestClassifier_ClassifyWithConfidence(t *testing.T) {
	c := NewClassifier()

	// Test that clear queries have higher confidence
	intent, confidence := c.ClassifyWithConfidence("SELECT * FROM users WHERE id = 1")
	if intent != IntentSQLQuery {
		t.Errorf("Expected sql_query intent, got %v", intent)
	}
	if confidence <= 0 {
		t.Errorf("Expected positive confidence, got %v", confidence)
	}

	// Test visualization query
	intent, confidence = c.ClassifyWithConfidence("Create a bar chart showing monthly sales")
	if intent != IntentVisualization {
		t.Errorf("Expected visualization intent, got %v", intent)
	}
	if confidence <= 0 {
		t.Errorf("Expected positive confidence, got %v", confidence)
	}
}

func TestNewClassifier(t *testing.T) {
	c := NewClassifier()
	if c == nil {
		t.Error("NewClassifier() returned nil")
	}
	if c.intentPrototypes == nil {
		t.Error("intentPrototypes is nil")
	}
	if len(c.intentPrototypes) != 3 {
		t.Errorf("Expected 3 intents, got %d", len(c.intentPrototypes))
	}
}
