package main

import (
	"context"
	_ "embed"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
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
	imageReader := llm.NewOpenAi(client)
	ccr := cards.NewLLMCustomCardReader(imageReader)
	card, err := ccr.ReadCard(ctx, `https://i.imgur.com/3KY9Lyt.png`)
	if err != nil {
		log.Fatal(fmt.Errorf(`read card: %w`, err))
	}
	fmt.Println(card)

	fmt.Println(`Read Card!`)
}
