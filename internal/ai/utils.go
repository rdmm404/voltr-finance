package ai

import (
	"encoding/json"
	"fmt"

	"google.golang.org/genai"
)

type LLMResponse interface {
	Text() string
	MarshalJSON() ([]byte, error)
}

func LLMResponseToString(response LLMResponse) string {
	jsonResponse, err := response.MarshalJSON()
	if err != nil {
		fmt.Printf("Something happened while marshaling LLM response, falling back to text %v", err)
		return response.Text()
	}

	return string(jsonResponse)
}

func LLMRequestToString(messages []*genai.Content, config *genai.GenerateContentConfig) (string, string) {
	contentJson, errContent := json.MarshalIndent(messages, "", "  ")
	configJson, errConfig := json.MarshalIndent(config, "", "  ")

	contentStr := string(contentJson)
	configStr := string(configJson)
	if errContent != nil {
		fmt.Printf("Something happened while marshaling LLM content, falling back to struct %v", errContent)
		contentStr = fmt.Sprintf("%+v", messages)
	}

	if errConfig != nil {
		fmt.Printf("Something happened while marshaling LLM config, falling back to struct %v", errConfig)
		configStr = fmt.Sprintf("%+v", config)
	}

	return contentStr, configStr
}
