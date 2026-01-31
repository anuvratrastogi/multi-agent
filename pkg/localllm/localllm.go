package localllm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Config holds configuration for the local LLM.
type Config struct {
	// BaseURL is the base URL of the local LLM server (e.g., "http://localhost:1234")
	BaseURL string
	// Model is the model name to use
	Model string
}

// LocalLLM implements model.LLM for OpenAI-compatible local LLM servers.
type LocalLLM struct {
	baseURL string
	model   string
	client  *http.Client
}

// New creates a new LocalLLM instance.
func New(cfg Config) *LocalLLM {
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")
	model := cfg.Model
	if model == "" {
		model = "local-model"
	}
	return &LocalLLM{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

// Name implements model.LLM.
func (l *LocalLLM) Name() string {
	return l.model
}

// OpenAI-compatible request/response types
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
	Tools       []toolDef     `json:"tools,omitempty"`
}

type chatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type toolDef struct {
	Type     string      `json:"type"`
	Function functionDef `json:"function"`
}

type functionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatResponse struct {
	ID      string   `json:"id"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type choice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// GenerateContent implements model.LLM.
func (l *LocalLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Convert ADK request to OpenAI format
		messages := l.convertToMessages(req)
		tools := l.convertToTools(req)

		chatReq := chatRequest{
			Model:    l.model,
			Messages: messages,
			Stream:   false, // For simplicity, we don't stream
			Tools:    tools,
		}

		// Add temperature if specified
		if req.Config != nil && req.Config.Temperature != nil {
			chatReq.Temperature = float64(*req.Config.Temperature)
		}

		reqBody, err := json.Marshal(chatReq)
		if err != nil {
			yield(nil, fmt.Errorf("failed to marshal request: %w", err))
			return
		}

		// DEBUG: Print request JSON
		fmt.Printf("\n🔎 [DEBUG] Sending to LLM:\n%s\n\n", string(reqBody))

		httpReq, err := http.NewRequestWithContext(ctx, "POST", l.baseURL+"/v1/chat/completions", bytes.NewReader(reqBody))
		if err != nil {
			yield(nil, fmt.Errorf("failed to create request: %w", err))
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := l.client.Do(httpReq)
		if err != nil {
			yield(nil, fmt.Errorf("failed to send request: %w", err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			yield(nil, fmt.Errorf("LLM request failed with status %d: %s", resp.StatusCode, string(body)))
			return
		}

		var chatResp chatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			yield(nil, fmt.Errorf("failed to decode response: %w", err))
			return
		}

		// Convert OpenAI response to ADK format
		llmResp := l.convertToLLMResponse(&chatResp)
		yield(llmResp, nil)
	}
}

func (l *LocalLLM) convertToMessages(req *model.LLMRequest) []chatMessage {
	var messages []chatMessage

	// Check for system instruction in config
	if req.Config != nil && req.Config.SystemInstruction != nil {
		var sysText string
		for _, part := range req.Config.SystemInstruction.Parts {
			if part.Text != "" {
				sysText += part.Text
			}
		}
		if sysText != "" {
			messages = append(messages, chatMessage{
				Role:    "system",
				Content: sysText,
			})
		}
	}

	// Convert contents to messages
	for _, content := range req.Contents {
		role := "user"
		if content.Role == "model" {
			role = "assistant"
		}

		var textContent string
		var funcCalls []toolCall
		var funcResponses []struct {
			id       string
			name     string
			response string
		}

		for _, part := range content.Parts {
			if part.Text != "" {
				textContent += part.Text
			}
			// Handle function calls from model
			if part.FunctionCall != nil {
				argsJSON, _ := json.Marshal(part.FunctionCall.Args)
				funcCalls = append(funcCalls, toolCall{
					ID:   part.FunctionCall.ID,
					Type: "function",
					Function: functionCall{
						Name:      part.FunctionCall.Name,
						Arguments: string(argsJSON),
					},
				})
			}
			// Handle function responses (tool results)
			if part.FunctionResponse != nil {
				responseJSON, _ := json.Marshal(part.FunctionResponse.Response)
				funcResponses = append(funcResponses, struct {
					id       string
					name     string
					response string
				}{
					id:       part.FunctionResponse.ID,
					name:     part.FunctionResponse.Name,
					response: string(responseJSON),
				})
			}
		}

		// Add text or function call message
		if len(funcCalls) > 0 {
			// This is a new assistant message with function calls - never merge tool calls
			messages = append(messages, chatMessage{
				Role:      "assistant",
				Content:   textContent,
				ToolCalls: funcCalls,
			})
		} else if textContent != "" {
			// Check if we can merge with previous message
			merged := false
			if len(messages) > 0 {
				lastIdx := len(messages) - 1
				lastMsg := &messages[lastIdx]

				// Only merge if roles match and neither has tool calls/ids (simple text messages)
				if lastMsg.Role == role && lastMsg.ToolCalls == nil && lastMsg.ToolCallID == "" {
					lastMsg.Content += "\n" + textContent
					merged = true
				}
			}

			if !merged {
				messages = append(messages, chatMessage{
					Role:    role,
					Content: textContent,
				})
			}
		}

		// Add function response messages (tool results)
		for _, fr := range funcResponses {
			messages = append(messages, chatMessage{
				Role:       "tool",
				Content:    fr.response,
				ToolCallID: fr.id,
			})
		}
	}

	// Post-processing: specific fix for Mistral/LM Studio
	// Ensure Tool messages are always followed by Assistant messages before the next User message
	var finalMessages []chatMessage
	for _, msg := range messages {
		// If we are about to add a User message, and the previous one was a Tool message
		if msg.Role == "user" && len(finalMessages) > 0 {
			lastMsg := finalMessages[len(finalMessages)-1]
			if lastMsg.Role == "tool" {
				// Inject a neutral assistant message to satisfy alternation
				finalMessages = append(finalMessages, chatMessage{
					Role:    "assistant",
					Content: "Action completed.",
				})
			}
		}
		finalMessages = append(finalMessages, msg)
	}

	return finalMessages
}

// knownToolSchemas provides fallback schemas for tools that don't have Parameters set.
// This is needed because ADK's functiontool doesn't populate genai.FunctionDeclaration.Parameters
// when converted for local LLM usage.
var knownToolSchemas = map[string]map[string]interface{}{
	"query_database": {
		"type": "object",
		"properties": map[string]interface{}{
			"sql": map[string]interface{}{
				"type":        "string",
				"description": "The SQL query to execute",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of rows to return (default: 100)",
			},
		},
		"required": []string{"sql"},
	},
	"get_schema": {
		"type": "object",
		"properties": map[string]interface{}{
			"table_name": map[string]interface{}{
				"type":        "string",
				"description": "The name of the table to get schema for",
			},
		},
		"required": []string{"table_name"},
	},
	"list_tables": {
		"type":       "object",
		"properties": map[string]interface{}{},
	},
	"describe_database": {
		"type":       "object",
		"properties": map[string]interface{}{},
	},
	"generate_chart": {
		"type": "object",
		"properties": map[string]interface{}{
			"chart_type": map[string]interface{}{
				"type":        "string",
				"description": "Type of chart: bar, line, pie, scatter",
			},
			"title": map[string]interface{}{
				"type":        "string",
				"description": "Chart title",
			},
			"data": map[string]interface{}{
				"type":        "string",
				"description": "JSON string containing the chart data",
			},
		},
		"required": []string{"chart_type", "title", "data"},
	},
}

func (l *LocalLLM) convertToTools(req *model.LLMRequest) []toolDef {
	var tools []toolDef

	if req.Config == nil || req.Config.Tools == nil {
		return tools
	}

	// Default empty parameters schema for OpenAI compatibility
	emptyParams := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}

	for _, t := range req.Config.Tools {
		if t.FunctionDeclarations != nil {
			for _, fd := range t.FunctionDeclarations {
				var params interface{} = emptyParams

				if fd.Parameters != nil {
					// Normalize the schema to ensure compatibility
					params = normalizeSchema(fd.Parameters)

					// DEBUG: Print parameter details
					paramJSON, _ := json.Marshal(params)
					fmt.Printf("🔎 [DEBUG] Tool %s normalized params: %s\n", fd.Name, string(paramJSON))
				} else if schema, ok := knownToolSchemas[fd.Name]; ok {
					// Use fallback schema for known tools
					params = schema

					// DEBUG: Print fallback
					paramJSON, _ := json.Marshal(params)
					fmt.Printf("🔎 [DEBUG] Tool %s using fallback schema: %s\n", fd.Name, string(paramJSON))
				}

				tools = append(tools, toolDef{
					Type: "function",
					Function: functionDef{
						Name:        fd.Name,
						Description: fd.Description,
						Parameters:  params,
					},
				})
			}
		}
	}

	return tools
}

// normalizeSchema ensures the schema is in standard OpenAI JSON schema format
// converting from potentially capitalized matching (genai) to lowercase
func normalizeSchema(params interface{}) interface{} {
	b, err := json.Marshal(params)
	if err != nil {
		return params
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return params
	}

	return fixSchemaMap(raw)
}

func fixSchemaMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range m {
		// Lowercase the key
		key := strings.ToLower(k)

		// Handle value based on connection
		switch val := v.(type) {
		case string:
			// Ensure types are lowercase
			if key == "type" {
				out[key] = strings.ToLower(val)
			} else {
				out[key] = val
			}
		case map[string]interface{}:
			if key == "properties" {
				newProps := make(map[string]interface{})
				for pk, pv := range val {
					if plot, ok := pv.(map[string]interface{}); ok {
						newProps[pk] = fixSchemaMap(plot)
					} else {
						newProps[pk] = pv
					}
				}
				out[key] = newProps
			} else {
				out[key] = fixSchemaMap(val)
			}
		default:
			out[key] = val
		}
	}
	return out
}

func (l *LocalLLM) convertToLLMResponse(chatResp *chatResponse) *model.LLMResponse {
	if len(chatResp.Choices) == 0 {
		return &model.LLMResponse{}
	}

	choice := chatResp.Choices[0]
	var parts []*genai.Part

	// Add text content
	if choice.Message.Content != "" {
		parts = append(parts, genai.NewPartFromText(choice.Message.Content))
	}

	// Add function calls
	for _, tc := range choice.Message.ToolCalls {
		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		parts = append(parts, genai.NewPartFromFunctionCall(tc.Function.Name, args))
	}

	content := &genai.Content{
		Role:  "model",
		Parts: parts,
	}

	return &model.LLMResponse{
		Content: content,
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     int32(chatResp.Usage.PromptTokens),
			CandidatesTokenCount: int32(chatResp.Usage.CompletionTokens),
			TotalTokenCount:      int32(chatResp.Usage.TotalTokens),
		},
	}
}
