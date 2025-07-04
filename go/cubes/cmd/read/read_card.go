package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/invopop/jsonschema"
	"github.com/jmoiron/sqlx"
	"github.com/mgdunn2/cube-datahub/cubes"
	"github.com/mgdunn2/cube-datahub/cubes/llm"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"log"
	"os"
)

func main() {
	ctx := context.Background()
	openAiApiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(option.WithAPIKey(openAiApiKey))
	classifier := llm.NewOpenAi(client)

	schema := GenerateSchema(cubes.ScryfallCard{})
	fmt.Println(schema)

	req := llm.Request{
		Prompt: `Read the custom magic the gathering card and fill in the card schema.
The ID field should be left as empty string. The Set should be "custom" and the release date can be anything.
Below is a description of how you should interpret the mana cost.

Canonical Mana Cost Format

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
		Url: `https://i.imgur.com/3KY9Lyt.png`,
		Schema: llm.ToolSchema{
			Name:        "card",
			Description: "A custom magic the gathering card.",
			Schema:      GenerateSchema(cubes.Card{}),
		},
	}

	res, err := classifier.Generate(ctx, req)
	if err != nil {
		log.Fatal(fmt.Errorf(`generate: %w`, err))
	}
	fmt.Println(res)
	var card cubes.ScryfallCard
	if err := json.Unmarshal([]byte(res), &card); err != nil {
		log.Fatal(fmt.Errorf(`unmarshal: %w`, err))
	}
	fmt.Println(card.ToCard())

	fmt.Println(`Read Card!`)
}

func mustDb(env string) *sqlx.DB {
	var connectionString string
	if env == "local" {
		connectionString = "root@tcp(127.0.0.1:3306)/cubes?parseTime=true&loc=America%2FNew_York"
	}
	db, err := sqlx.Open("mysql", connectionString)
	if err != nil {
		log.Fatal(fmt.Errorf("connect MySQL: %w", err))
	}
	return db
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
