package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

type ImageReader interface {
	Generate(ctx context.Context, req Request) (string, error)
}

type OpenAi struct {
	client openai.Client
}

func NewOpenAi(client openai.Client) OpenAi {
	return OpenAi{client: client}
}

type Request struct {
	Prompt    string     `json:"prompt"`
	ImageURL  string     `json:"imageUrl"`
	ImageByes []byte     `json:"imageData"`
	Schema    ToolSchema `json:"schema"`
}

type ToolSchema struct {
	Name        string
	Description string
	Schema      map[string]any
}

func (o OpenAi) Generate(ctx context.Context, req Request) (string, error) {
	var imageContent openai.ChatCompletionContentPartUnionParam
	if req.ImageByes != nil && len(req.ImageByes) > 0 {
		imageContent = openai.ChatCompletionContentPartUnionParam{
			OfImageURL: &openai.ChatCompletionContentPartImageParam{
				ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
					URL:    fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(req.ImageByes)),
					Detail: "high",
				},
			},
		}
	} else if req.ImageURL != "" {
		imageContent = openai.ChatCompletionContentPartUnionParam{
			OfImageURL: &openai.ChatCompletionContentPartImageParam{
				ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
					URL:    req.ImageURL,
					Detail: "high",
				},
			},
		}
	} else {
		return "", errors.New("no image provided")
	}
	res, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: "gpt-4o",
		Messages: []openai.ChatCompletionMessageParamUnion{
			{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfArrayOfContentParts: []openai.ChatCompletionContentPartTextParam{
							{
								Text: "You are a helpful assistant whose job it is to assist with image classification. Use the provided tools to help classify the images",
							},
						},
					},
				},
			},
			{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfArrayOfContentParts: []openai.ChatCompletionContentPartUnionParam{
							{
								OfText: &openai.ChatCompletionContentPartTextParam{
									Text: req.Prompt,
								},
							},
							// Add the image content
							imageContent,
						},
					},
				},
			},
		},
		Tools: []openai.ChatCompletionToolParam{
			{
				Function: openai.FunctionDefinitionParam{
					Name:        req.Schema.Name,
					Description: param.Opt[string]{Value: req.Schema.Description},
					Strict:      param.Opt[bool]{Value: true},
					Parameters:  req.Schema.Schema,
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("generate: %w", err)
	}
	for _, choice := range res.Choices {
		for _, tool := range choice.Message.ToolCalls {
			return tool.Function.Arguments, nil
		}
	}
	return "", errors.New("no tool call found")
}

func GenerateSchema(v any) map[string]any {
	r := new(jsonschema.Reflector)
	schema := r.Reflect(v)

	raw, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}

	var top map[string]any
	if err := json.Unmarshal(raw, &top); err != nil {
		panic(err)
	}

	// Navigate to the definition referenced by $ref
	ref := top["$ref"].(string) // e.g., "#/$defs/Card"
	defs := top["$defs"].(map[string]any)
	defKey := ref[len("#/$defs/"):] // "Card"
	cardSchema := defs[defKey].(map[string]any)

	return cardSchema
}
