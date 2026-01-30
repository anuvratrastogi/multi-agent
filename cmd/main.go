package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/anuvratrastogi/multi-agent/config"
	"github.com/anuvratrastogi/multi-agent/internal/agents/chart"
	"github.com/anuvratrastogi/multi-agent/internal/agents/manager"
	sqlagent "github.com/anuvratrastogi/multi-agent/internal/agents/sql"
	"github.com/anuvratrastogi/multi-agent/pkg/localllm"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Load configuration
	cfg := config.New()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	fmt.Println("🤖 Multi-Agent System")
	fmt.Println("=====================")

	// Initialize LLM based on provider
	var llm model.LLM
	var err error

	if cfg.IsLocalLLM() {
		fmt.Printf("🔧 Using Local LLM: %s\n", cfg.LocalLLMURL)
		fmt.Printf("   Model: %s\n", cfg.Model)
		llm = localllm.New(localllm.Config{
			BaseURL: cfg.LocalLLMURL,
			Model:   cfg.Model,
		})
	} else {
		fmt.Printf("🔧 Using Gemini: %s\n", cfg.Model)
		llm, err = gemini.NewModel(ctx, cfg.Model, &genai.ClientConfig{
			APIKey: cfg.GoogleAPIKey,
		})
		if err != nil {
			log.Fatalf("Failed to initialize Gemini model: %v", err)
		}
	}
	fmt.Println()

	// Initialize database client
	fmt.Println("📊 Connecting to PostgreSQL...")
	dbClient, err := sqlagent.NewDirectMCPClient(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbClient.Close()
	fmt.Println("✅ Database connected")

	// Fetch database schema for SQL agent
	fmt.Println("📋 Loading database schema...")
	dbSchema, err := dbClient.DescribeDatabase(ctx)
	if err != nil {
		log.Printf("⚠️  Warning: Could not load schema: %v", err)
		dbSchema = ""
	} else {
		fmt.Println("✅ Schema loaded")
	}

	// Create tools for SQL agent
	sqlTools, err := sqlagent.CreateMCPTools(dbClient)
	if err != nil {
		log.Fatalf("Failed to create SQL tools: %v", err)
	}

	// Initialize SQL Agent with schema
	fmt.Println("🔧 Initializing SQL Agent...")
	sqlAgent, err := sqlagent.New(sqlagent.Config{
		Model:          llm,
		Tools:          sqlTools,
		DatabaseSchema: dbSchema,
	})
	if err != nil {
		log.Fatalf("Failed to create SQL agent: %v", err)
	}
	fmt.Println("✅ SQL Agent ready")

	// Initialize Chart Agent
	fmt.Println("📈 Initializing Chart Agent...")
	chartAgent, err := chart.New(chart.Config{
		Model: llm,
	})
	if err != nil {
		log.Fatalf("Failed to create Chart agent: %v", err)
	}
	fmt.Println("✅ Chart Agent ready")

	// Initialize Manager Agent
	fmt.Println("👔 Initializing Manager Agent...")
	managerAgent, err := manager.New(manager.Config{
		Model:      llm,
		SQLAgent:   sqlAgent,
		ChartAgent: chartAgent,
	})
	if err != nil {
		log.Fatalf("Failed to create Manager agent: %v", err)
	}
	fmt.Println("✅ Manager Agent ready")

	// Create the session service and session
	fmt.Println("🏃 Creating ADK Runner...")
	sessionService := session.InMemoryService()

	adkRunner, err := runner.New(runner.Config{
		AppName:        "multi-agent",
		Agent:          managerAgent,
		SessionService: sessionService,
	})
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}
	fmt.Println("✅ Runner ready")
	fmt.Println()

	// Start interactive REPL
	fmt.Println("Type your queries below. Type 'quit' or 'exit' to stop.")
	fmt.Println("Examples:")
	fmt.Println("  - Show me all tables in the database")
	fmt.Println("  - How many orders are there per month?")
	fmt.Println("  - Create a bar chart of sales by month")
	fmt.Println()

	sessionID := "session-1"
	userID := "user-1"

	// Create the session first
	_, err = sessionService.Create(ctx, &session.CreateRequest{
		AppName:   "multi-agent",
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "quit" || input == "exit" {
			fmt.Println("Goodbye! 👋")
			break
		}

		// Show intent classification
		result, _ := managerAgent.ProcessQuery(ctx, input)
		fmt.Printf("\n📋 Intent: %s (confidence: %.2f)\n", result.ClassifiedIntent, result.Confidence)
		fmt.Printf("🔄 Workflow: %s\n", result.Workflow)
		fmt.Printf("🤖 Agents: %s\n\n", strings.Join(result.AgentsUsed, " → "))

		// Create user message
		userMsg := genai.NewContentFromText(input, genai.RoleUser)

		// Execute through ADK runner
		fmt.Println("⏳ Processing...")
		var responseText strings.Builder

		for event, err := range adkRunner.Run(ctx, userID, sessionID, userMsg, agent.RunConfig{}) {
			if err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				break
			}

			// Process the event
			if event != nil && event.LLMResponse.Content != nil {
				// Check for tool calls
				for _, part := range event.LLMResponse.Content.Parts {
					if part.FunctionCall != nil {
						fmt.Printf("  🔧 [AGENT] Calling tool: %s\n", part.FunctionCall.Name)
					}
					if part.Text != "" {
						responseText.WriteString(part.Text)
					}
				}
			}
		}

		// Print the response
		if responseText.Len() > 0 {
			fmt.Printf("\n🤖 Agent: %s\n\n", responseText.String())
		} else {
			fmt.Println("\n💡 No response generated.\n")
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}
}
