package tools

import "google.golang.org/genai"

type Tool interface {
	Name() string
	Description() string
	Parameters() *genai.Schema
	Call(functionCall *genai.FunctionCall) *genai.FunctionResponse
}

var allTools = []Tool{
	SaveTransactionsTool{},
}
var toolsByName = make(map[string]Tool)
var genaiTools []*genai.Tool

func init() {
	for _, tool := range allTools {
		toolsByName[tool.Name()] = tool

		genaiTools = append(genaiTools, &genai.Tool{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        tool.Name(),
					Description: tool.Description(),
					Parameters:  tool.Parameters(),
				},
			},
		})
	}
}

func GetTools() []Tool {
	return allTools
}

func GetToolByName(name string) (Tool, bool) {
	tool, ok := toolsByName[name]
	return tool, ok
}

func GetGenaiTools() []*genai.Tool {
	return genaiTools
}
