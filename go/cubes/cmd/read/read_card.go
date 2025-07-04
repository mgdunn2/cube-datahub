package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/invopop/jsonschema"
	"github.com/jmoiron/sqlx"
	"github.com/mgdunn2/cube-datahub/cubes/cards"
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
	ccr := cards.NewLLMCustomCardReader(classifier)
	card, err := ccr.ReadCard(ctx, `https://i.imgur.com/3KY9Lyt.png`)
	if err != nil {
		log.Fatal(fmt.Errorf(`read card: %w`, err))
	}
	fmt.Println(card)

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
