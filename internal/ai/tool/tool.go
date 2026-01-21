package tool

import (
	"fmt"
	"log/slog"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/transaction"
	"rdmm404/voltr-finance/internal/utils"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type Tool interface {
	Name() string
	Description() string
	Create(tp *ToolProvider) ai.Tool
}

type ToolProvider struct {
	allTools []ai.Tool
	deps     *ToolDependencies
}

type ToolDependencies struct {
	Ts *transaction.TransactionService
	ReadOnlyDB sqlc.DBTX
	Genkit *genkit.Genkit
}

var toolFactories = []func(deps *ToolDependencies) (Tool, error){
	NewSaveTransactionsTool,
	NewGetTransactionsTool,
	NewUpdateTransactionsByIdTool,
	NewQueryTool,
}

func NewToolProvider(deps *ToolDependencies) *ToolProvider {
	return &ToolProvider{deps: deps}
}

func (tp *ToolProvider) Init() error {
	for _, toolFactory := range toolFactories {
		tool, err := toolFactory(tp.deps)
		if err != nil {
			return fmt.Errorf("error while creating tool - %w", err)
		}
		tp.allTools = append(tp.allTools, tool.Create(tp))
	}
	return nil
}

func (tp *ToolProvider) GetAvailableTools() []ai.ToolRef {
	var toolRefs []ai.ToolRef
	for _, tool := range tp.allTools {
		toolRefs = append(toolRefs, tool)
	}
	return toolRefs
}

func DefineTool[I any, O any](
	tp *ToolProvider,
	tool Tool,
	handler ai.ToolFunc[I, O],
) ai.Tool {
	return genkit.DefineTool(
		tp.deps.Genkit,
		tool.Name(),
		tool.Description(),
		func(ctx *ai.ToolContext, input I) (O, error) {
			slog.Debug("Tool called", "name", tool.Name(), "input", utils.JsonMarshalIgnore(input))
			res, err := handler(ctx, input)
			slog.Debug("Tool response received", "tool", tool.Name(), "response", res)
			if err != nil {
				return res, err
			}
			return res, err
		},
	)

}
