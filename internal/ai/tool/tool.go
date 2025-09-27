package tool

import (
	"fmt"
	"rdmm404/voltr-finance/internal/transaction"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type Tool interface {
	Name() string
	Description() string
	Create(g *genkit.Genkit, tp *ToolProvider) ai.Tool
}

type ToolProvider struct {
	allTools      []ai.Tool
	deps *ToolDependencies
}

type ToolDependencies struct {
	Ts *transaction.TransactionService
}


var toolFactories = []func(deps *ToolDependencies) (Tool, error) {
	NewSaveTransactionsTool,
}

func NewToolProvider(deps *ToolDependencies) *ToolProvider {
	return &ToolProvider{deps: deps}
}

func (tp *ToolProvider) Init(g *genkit.Genkit) error {
	for _, toolFactory := range toolFactories {
		tool, err := toolFactory(tp.deps)
		if err != nil {
			return fmt.Errorf("error while creating tool - %w", err)
		}
		tp.allTools = append(tp.allTools, tool.Create(g, tp))
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
	g *genkit.Genkit,
	tool Tool,
	handler ai.ToolFunc[I, O],
) ai.Tool {
	return genkit.DefineTool(
		g,
		tool.Name(),
		tool.Description(),
		func(ctx *ai.ToolContext, input I) (O, error) {
			res, err := handler(ctx, input)
			if err != nil {
				return res, err
			}
			return res, err
		},
	)

}