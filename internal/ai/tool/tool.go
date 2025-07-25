package tool

import (
	"fmt"
	"rdmm404/voltr-finance/internal/transaction"

	"google.golang.org/genai"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() *genai.Schema
	Call(functionCall *genai.FunctionCall, deps *ToolDependencies) *genai.FunctionResponse
}

type ToolProvider struct {
	toolsByName     map[string]Tool
	genaiTools      []*genai.Tool
	deps *ToolDependencies
}

type ToolDependencies struct {
	Ts *transaction.TransactionService
}

var allTools = []Tool{
	SaveTransactionsTool{},
}

func NewToolProvider(deps *ToolDependencies) *ToolProvider {
	tp := ToolProvider{deps: deps}
	tp.toolsByName = make(map[string]Tool, 0)

	for _, tool := range allTools {
		tp.toolsByName[tool.Name()] = tool

		tp.genaiTools = append(tp.genaiTools, &genai.Tool{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        tool.Name(),
					Description: tool.Description(),
					Parameters:  tool.Parameters(),
				},
			},
		})
	}

	return &tp
}

func (tp *ToolProvider) GetToolByName(name string) (Tool, bool) {
	fmt.Printf("getting tool by name %v\n", name)
	fmt.Printf("tools by name %+v\n", tp.toolsByName)
	tool, ok := tp.toolsByName[name]
	return tool, ok
}

func (tp *ToolProvider) GetGenaiTools() []*genai.Tool {
	return tp.genaiTools
}

func (tp *ToolProvider) ExecuteToolCall(call *genai.FunctionCall) *genai.FunctionResponse {
	tool, ok := tp.GetToolByName(call.Name)

	if !ok {
		return &genai.FunctionResponse{
			ID:   call.ID,
			Name: call.Name,
			Response: map[string]any{
				"error": "Function with name " + call.Name + "Was not found",
			},
		}
	}

	return tool.Call(call, tp.deps)
}
