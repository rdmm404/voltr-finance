package ai

import (
	"encoding/json"
	"fmt"

	"google.golang.org/genai"
)

type LLMResponse interface {
	Text() string
}

func LLMResponseToString(response LLMResponse) string {
	// jsonResponse, err := json.MarshalIndent(response, "", " ")
	// if err != nil {
	// 	fmt.Printf("Something happened while marshaling LLM response, falling back to text %v", err)
	// 	return response.Text()
	// }

	return fmt.Sprintf("%+v", response)
}

func LLMRequestToString(messages []*genai.Content, config *genai.GenerateContentConfig) (string, string) {
	// contentJson, errContent := json.MarshalIndent(messages, "", "  ")
	configJson, errConfig := json.MarshalIndent(config, "", "  ")

	configStr := string(configJson)
	// contentStr := string(contentJson)
	// if errContent != nil {
	// 	fmt.Printf("Something happened while marshaling LLM content, falling back to struct %v", errContent)
	// }

	if errConfig != nil {
		fmt.Printf("Something happened while marshaling LLM config, falling back to struct %v", errConfig)
		configStr = fmt.Sprintf("%+v", config)
	}
	contentStr := fmt.Sprintf("%+v", messages)

	return contentStr, configStr
}
