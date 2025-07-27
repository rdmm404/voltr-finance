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

func LLMRequestToString(messages []*genai.Content) string {
	return ContentSlice(messages).String()
}

type ContentSlice []*genai.Content

func (cs ContentSlice) String() string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, content := range cs {
		if i > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString("genai.Content{ ")
		sb.WriteString(fmt.Sprintf("Parts: %+v, ", PartSlice(content.Parts)))
		sb.WriteString(fmt.Sprintf("Role: %v }", content.Role))

	}
	sb.WriteString("]")
	return sb.String()
}

type PartSlice []*genai.Part

func (ps PartSlice) String() string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, part := range ps {
		if i > 0 {
			sb.WriteString(", ")
		}

		partJson, err := json.Marshal((MyPart(*part)))

		if err != nil {
			sb.WriteString(fmt.Sprintf("%+v", *part))
			continue
		}

		sb.WriteString(string(partJson))
	}
	sb.WriteString("]")
	return sb.String()
}

type MyPart genai.Part

type MyBlob struct {
	genai.Blob
	FormattedData string `json:"data,omitempty"`
}

func (p MyPart) MarshalJSON() ([]byte, error){
	type Alias MyPart
	if p.InlineData != nil && len(p.InlineData.Data) > 1 {
		return json.Marshal(&struct {
			Alias
			FormattedInlineData MyBlob `json:"inlineData,omitempty"`
		}{
			Alias: (Alias)(p),
			FormattedInlineData: MyBlob{
				Blob: *p.InlineData,
				FormattedData: fmt.Sprintf("%v bytes", len(p.InlineData.Data)),
			},
		})
	}

	return json.Marshal((Alias)(p))
}