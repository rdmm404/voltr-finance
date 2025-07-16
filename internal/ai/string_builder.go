package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

type LLMResponse interface {
	Text() string
}

func LLMResponseToString(response LLMResponse) string {
	jsonResponse, err := json.MarshalIndent(response, "", " ")
	if err != nil {
		fmt.Printf("Something happened while marshaling LLM response, falling back to text %v", err)
		return response.Text()
	}

	return string(jsonResponse)
}

func LLMRequestToString(messages []*genai.Content, config *genai.GenerateContentConfig) (string, string) {
	configJson, errConfig := json.MarshalIndent(config, "", "  ")

	configStr := string(configJson)
	contentStr := ContentSlice(messages).String()

	if errConfig != nil {
		fmt.Printf("Something happened while marshaling LLM config, falling bac`k to struct %v", errConfig)
		configStr = fmt.Sprintf("%+v", config)
	}

	return contentStr, configStr
}

type ContentSlice []*genai.Content

func (cs ContentSlice) String() string {
	var sb strings.Builder
	fmt.Println("content string was called")
	sb.WriteString("[")
	for i, content := range cs {
		if i > 0 {
			sb.WriteString(", ")
		}

		parts := PartSlice(content.Parts).String()
		sb.WriteString("genai.Content{")
		sb.WriteString(fmt.Sprintf("Parts: %+v", parts))
		sb.WriteString(fmt.Sprintf("Role: %v}", content.Role))

	}
	sb.WriteString("]")
	return sb.String()
}

type PartSlice []*genai.Part

func (ps PartSlice) String() string {
	var sb strings.Builder
	fmt.Println("part string was called")
	sb.WriteString("[")
	for i, part := range ps {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%+v", *part))
	}
	sb.WriteString("]")
	return sb.String()
}
