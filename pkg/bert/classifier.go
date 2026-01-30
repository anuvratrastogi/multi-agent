package bert

import (
	"math"
	"strings"
)

// Intent represents the classified intent type.
type Intent string

const (
	IntentSQLQuery      Intent = "sql_query"
	IntentVisualization Intent = "visualization"
	IntentGeneral       Intent = "general"
)

// Classifier classifies user queries into intents.
// Uses a combination of keyword matching and semantic similarity.
type Classifier struct {
	// Prototype embeddings for each intent
	intentPrototypes map[Intent][]string
}

// NewClassifier creates a new intent classifier.
func NewClassifier() *Classifier {
	return &Classifier{
		intentPrototypes: map[Intent][]string{
			IntentSQLQuery: {
				"query", "select", "fetch", "get", "show", "list", "find",
				"database", "table", "data", "rows", "records", "sql",
				"where", "from", "join", "count", "sum", "average",
				"filter", "search", "lookup", "retrieve",
			},
			IntentVisualization: {
				"chart", "graph", "plot", "visualize", "visualization",
				"bar", "line", "pie", "scatter", "histogram",
				"display", "render", "draw", "show chart", "create chart",
				"trend", "comparison", "distribution",
			},
			IntentGeneral: {
				"help", "how", "what", "explain", "describe",
				"information", "about", "tell me",
			},
		},
	}
}

// Classify determines the intent of a user query.
func (c *Classifier) Classify(query string) Intent {
	queryLower := strings.ToLower(query)
	words := strings.Fields(queryLower)

	scores := make(map[Intent]float64)

	// Calculate keyword match scores
	for intent, keywords := range c.intentPrototypes {
		score := 0.0
		for _, keyword := range keywords {
			if strings.Contains(queryLower, keyword) {
				// Weight exact word matches higher
				for _, word := range words {
					if word == keyword {
						score += 2.0
					} else if strings.Contains(word, keyword) || strings.Contains(keyword, word) {
						score += 1.0
					}
				}
				// Substring match
				if score == 0 {
					score += 0.5
				}
			}
		}
		// Normalize by keyword count
		scores[intent] = score / float64(len(keywords))
	}

	// Apply heuristic rules for better classification
	scores = c.applyHeuristics(queryLower, scores)

	// Find highest scoring intent
	maxScore := 0.0
	bestIntent := IntentGeneral

	for intent, score := range scores {
		if score > maxScore {
			maxScore = score
			bestIntent = intent
		}
	}

	// If the score is too low, default to general
	if maxScore < 0.1 {
		return IntentGeneral
	}

	return bestIntent
}

// applyHeuristics applies additional rules to improve classification.
func (c *Classifier) applyHeuristics(query string, scores map[Intent]float64) map[Intent]float64 {
	// If query explicitly mentions charts/graphs, boost visualization
	if containsAny(query, []string{"chart", "graph", "plot", "visualize"}) {
		scores[IntentVisualization] += 1.0
	}

	// If query asks about database structure, boost SQL
	if containsAny(query, []string{"table", "schema", "column", "database"}) {
		scores[IntentSQLQuery] += 0.5
	}

	// Sequential workflow detection: if query mentions both data and visualization
	if containsAny(query, []string{"show", "display"}) && containsAny(query, []string{"chart", "graph"}) {
		// This might be a combined query - visualization takes priority
		scores[IntentVisualization] += 0.5
	}

	// If it's a question about data, it's likely SQL
	if strings.HasPrefix(query, "how many") || strings.HasPrefix(query, "what is") {
		if containsAny(query, []string{"in the database", "in the table", "records", "rows"}) {
			scores[IntentSQLQuery] += 0.5
		}
	}

	return scores
}

// containsAny checks if the string contains any of the substrings.
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// ClassifyWithConfidence returns the intent along with a confidence score.
func (c *Classifier) ClassifyWithConfidence(query string) (Intent, float64) {
	queryLower := strings.ToLower(query)
	words := strings.Fields(queryLower)

	scores := make(map[Intent]float64)

	// Calculate keyword match scores
	for intent, keywords := range c.intentPrototypes {
		score := 0.0
		for _, keyword := range keywords {
			if strings.Contains(queryLower, keyword) {
				for _, word := range words {
					if word == keyword {
						score += 2.0
					} else if strings.Contains(word, keyword) || strings.Contains(keyword, word) {
						score += 1.0
					}
				}
				if score == 0 {
					score += 0.5
				}
			}
		}
		scores[intent] = score / float64(len(keywords))
	}

	scores = c.applyHeuristics(queryLower, scores)

	// Find highest and second highest
	maxScore := 0.0
	secondScore := 0.0
	bestIntent := IntentGeneral

	for intent, score := range scores {
		if score > maxScore {
			secondScore = maxScore
			maxScore = score
			bestIntent = intent
		} else if score > secondScore {
			secondScore = score
		}
	}

	// Calculate confidence based on margin between top two scores
	confidence := 0.0
	if maxScore > 0 {
		if secondScore > 0 {
			confidence = (maxScore - secondScore) / maxScore
		} else {
			confidence = math.Min(maxScore, 1.0)
		}
	}

	if maxScore < 0.1 {
		return IntentGeneral, 0.0
	}

	return bestIntent, confidence
}
