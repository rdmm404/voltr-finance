package tool

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"rdmm404/voltr-finance/internal/database"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/utils"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
)

const systemPrompt string = `
You are an expert at PostgreSQL, skilled in building accurate queries based on natural language inputs.
You will be provided with a SQL schema and a natural language prompt. Your task is to do your best to convert this prompt into a valid PostgreSQL query that matches
the desired output provided in the prompt. Your query MUST abide to the provided schema and be valid PostgreSQL syntactically. After the query is executed, you will be
provided with the output, which you'll have to reason about and decide if it satisfies the provided prompt. If not, you can make improvements to your query and execute it again.

Some general guidelines:
- Always use well-defined columns when querying, avoid using 'SELECT *'.
- Only use the columns that are strictly necessary to satisfy the query. Do not add unnecessary columns which were not requested.
- Be mindful of performance, prefer using indexes when possible.
- You have a total of %v attempts to fulfill this request. If you exceed this number of attempts, your last provided query will be used.`

var tablesForSchema []string = []string{"transaction", "household", "user", "household_user"}

type queryInterruptMetadata struct {
	Query string
}

type QueryInput struct {
	Query string
}

type QueryTool struct {
	deps    *ToolDependencies
	tools   []ai.ToolRef
	queries *sqlc.Queries
}

func NewQueryTool(deps *ToolDependencies) (Tool, error) {
	if deps.ReadOnlyDB == nil {
		return &QueryTool{}, fmt.Errorf("read only db is required as a dependency")
	}
	qt := &QueryTool{deps: deps, queries: sqlc.New(deps.ReadOnlyDB)}
	qt.tools = append(qt.tools, qt.runQueryTool(), qt.confirmQueryTool())
	return qt, nil
}

func (qt *QueryTool) Name() string {
	return "FilterTransactions"
}
func (qt *QueryTool) Description() string {
	return `Use this tool to get any transaction data you need, in natural language.
		This tool takes a single input that is a string, which should be your query in simple text.
		Then another agent will process it, get the data you requested and provide it to you. Use this whenever the user asks for any data regarding their transactions.
		Some examples on how to use this tool:
		Example 1:
		- User: What are my highest spending transactions for the past month?
		- Agent: FilterTransactions('List all transactions since 2025-01-01, ordered by transaction amount in descending order, return the first 10 results')
		Example 2:
		- User: Do I have any transactions for the past weekend?
		- Agent: FilterTransactions('List all transactions with transaction date from 2025-01-01 to 2025-01-03 ordered by most recent')
		Example 3:
		- User: Hey do i have any recent paypal transactions?
		- Agent: FilterTransactions('Find transactions with description or notes containing the term "paypal", order by most recent, return top 10 results')
		Example 4:
		- User: How much has each user in my household spent this month?
		- Agent: FilterTransactions('Find all transactions from 2025-01-01 to date, aggregate the amount by user and return the total amount per user')`
}

func (qt *QueryTool) Create(tp *ToolProvider) ai.Tool {
	return DefineTool(tp, qt, qt.execute)
}

// TODO find a way to not repeat execution of confirmed query
func (qt *QueryTool) execute(ctx *ai.ToolContext, input QueryInput) (string, error) {
	dbSchema, err := database.InspectTables(ctx, qt.queries, tablesForSchema)
	if err != nil {
		return "", fmt.Errorf("error while inspecting db schema: %w", err)
	}

	dbSchemaJson, err := json.Marshal(dbSchema)
	if err != nil {
		return "", fmt.Errorf("error while converting db schema to json: %w", err)
	}

	maxTurns := 10
	resp, err := genkit.Generate(
		ctx, qt.deps.Genkit,
		ai.WithSystem(systemPrompt, maxTurns),
		ai.WithMaxTurns(maxTurns),
		ai.WithTools(qt.tools...),
		ai.WithPrompt("The database schema is: %s\nHere is the prompt to query: %s", dbSchemaJson, input.Query),
	)

	slog.Debug("QueryTool: LLM response", "response", utils.JsonMarshalIgnore(resp))

	var query string
	if err != nil {
		slog.Error("QueryTool: error while calling LLM", "error", err)

		var genkitErr *core.GenkitError
		if !errors.As(err, &genkitErr) {
			return fmt.Sprintf("unexpected error received while calling agent: %q", err), nil
		}

		if genkitErr.Status != core.ABORTED {
			return fmt.Sprintf("unexpected error received while calling agent: %q", err), nil
		}

		slog.Warn("QueryTool: max iterations exceeded", "error", err)
		// TODO handle this
		query = "SELECT 1"
	}

	if resp.FinishReason != ai.FinishReasonInterrupted {
		slog.Error("QueryTool: unexpected interruption received", "response", utils.JsonMarshalIgnore(resp))
		return fmt.Sprintf("unexpected response received while calling agent: %v", resp), nil
	}

	for _, interrupt := range resp.Interrupts() {
		if meta, ok := ai.InterruptAs[queryInterruptMetadata](interrupt); ok {
			query = meta.Query
		}
	}

	return qt.executeQueryForLLM(ctx, query)
}

func (qt *QueryTool) runQueryTool() ai.Tool {
	return genkit.DefineTool(
		qt.deps.Genkit, "RunQuery", "Use this tool to run your query and see its results.",
		func(ctx *ai.ToolContext, input QueryInput) (string, error) {
			return qt.executeQueryForLLM(ctx, input.Query)
		},
	)
}

func (qt *QueryTool) confirmQueryTool() ai.Tool {
	return genkit.DefineTool(
		qt.deps.Genkit, "ConfirmQuery", "Use this tool to confirm that this is the correct query. It will then be executed and its results returned to the user.",
		func(ctx *ai.ToolContext, input QueryInput) (string, error) {
			return "", ai.InterruptWith(ctx, queryInterruptMetadata{Query: input.Query})
		},
	)
}

func (qt *QueryTool) executeQueryForLLM(ctx context.Context, query string) (string, error) {
	results, err := database.RunRawQuery(ctx, qt.deps.ReadOnlyDB, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "No rows returned.", nil
		}
		slog.Warn("QueryTool: error executing sql", "error", err)
		return fmt.Sprintf("Error while executing sql: %q", err), nil
	}
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("error converting result into JSON: %w", err)
	}

	return string(resultsJSON), nil
}
