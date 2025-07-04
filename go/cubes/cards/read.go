package cards

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/invopop/jsonschema"
	"github.com/mgdunn2/cube-datahub/cubes"
	"github.com/mgdunn2/cube-datahub/cubes/llm"
)

type CustomCardReader interface {
	ReadCard(ctx context.Context, imageURL string) (cubes.Card, error)
}

type LLMCustomCardReader struct {
	imageReader llm.ImageReader
}

func NewLLMCustomCardReader(imageReader llm.ImageReader) *LLMCustomCardReader {
	return &LLMCustomCardReader{
		imageReader: imageReader,
	}
}

func (l *LLMCustomCardReader) ReadCard(ctx context.Context, imageURL string) (cubes.Card, error) {
	req := llm.Request{
		Prompt: `Read the custom magic the gathering card and fill in the card schema.
The ID field should be left as empty string. The Set should be "custom" and the release date can be anything.

## Color Field

The color field should contain an array of the colors a card belongs to with the following mappings:
* White: 'W'
* Blue: 'U'
* Black: 'B'
* Red: 'R'
* Green: 'G'

Below is a description of how you should interpret the mana cost.

## Canonical Mana Cost Format

The mana cost of a Magic: The Gathering card is represented as a string of curly-braced symbols, describing the mana required to cast the card. Each individual mana symbol is enclosed in {} and the full cost is written as a sequence of such tokens.
🔢 Examples:

    {2}{U} = two colorless mana and one blue mana

    {W}{W}{U}{U} = two white, two blue

    {X}{R} = variable mana (X) and one red

    {1}{G}{G} = one colorless, two green

🎨 Valid Symbols:

    Color mana: {W}, {U}, {B}, {R}, {G}

    Colorless mana: {C}

    Numeric mana (generic): {0} through {20} (and beyond)

    Hybrid mana: {W/U}, {2/R}, {G/P} — using a slash / between options

    Snow mana: {S}

    Phyrexian mana: {W/P}, {U/P}, etc.

    Half mana (Un-sets only): {H} (rare, can be ignored unless relevant)

🧬 Structure:

    The full mana cost is a single string: each symbol starts with { and ends with }

    There are no spaces or delimiters outside the braces

    Symbols appear in casting order, but not necessarily in a canonical sorted order

    No additional context or formatting should be included — only the curly-brace tokens

✅ Valid Examples:

    {3}{G}

    {B}{B}{R}

    {1}{W/U}{W/U}

    {X}{G}{G}

    {S}{S}{G}`,
		Url: imageURL,
		Schema: llm.ToolSchema{
			Name:        "card",
			Description: "A custom magic the gathering card.",
			Schema:      GenerateSchema(cubes.LLMCardSchema{}),
		},
	}

	res, err := l.imageReader.Generate(ctx, req)
	if err != nil {
		return cubes.Card{}, fmt.Errorf(`generate: %w`, err)
	}

	var llmCard cubes.LLMCardSchema
	if err := json.Unmarshal([]byte(res), &llmCard); err != nil {
		return cubes.Card{}, fmt.Errorf(`unmarshal: %w`, err)
	}
	card, err := llmCard.ToCard()
	if err != nil {
		return cubes.Card{}, fmt.Errorf(`to card: %w`, err)
	}
	return card, nil
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
