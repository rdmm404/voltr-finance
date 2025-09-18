package tool

import (
	"rdmm404/voltr-finance/internal/transaction"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type Tool interface {
	Name() string
	Description() string
	Create(g *genkit.Genkit, deps *ToolDependencies) ai.Tool
}

type ToolProvider struct {
	allTools      []ai.ToolRef
	deps *ToolDependencies
}

type ToolDependencies struct {
	Ts *transaction.TransactionService
}


var allTools = []Tool{
	SaveTransactionsTool{},
}

func NewToolProvider(deps *ToolDependencies) *ToolProvider {
	return &ToolProvider{deps: deps}
}

func (tp *ToolProvider) Init(g *genkit.Genkit) {
	for _, tool := range allTools {
		tp.allTools = append(tp.allTools, tool.Create(g, tp.deps))
	}
}

func (tp *ToolProvider) GetAvailableTools() []ai.ToolRef {
	return tp.allTools
}
