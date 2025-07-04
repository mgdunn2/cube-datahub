package llm

import (
	"context"
	"errors"
	"fmt"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

type LLMImageReader interface {
	Generate(ctx context.Context, req Request) (string, error)
}

type OpenAi struct {
	client openai.Client
}

func NewOpenAi(client openai.Client) OpenAi {
	return OpenAi{client: client}
}

type Request struct {
	Prompt string `json:"prompt"`
	Url    string `json:"url"`

	Schema ToolSchema `json:"schema"`
}

type ToolSchema struct {
	Name        string
	Description string
	Schema      map[string]any
}

func (o *OpenAi) Generate(ctx context.Context, req Request) (string, error) {
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
							{
								OfImageURL: &openai.ChatCompletionContentPartImageParam{
									ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
										URL:    req.Url,
										Detail: "high",
									},
								},
							},
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
