package main

import (
	"context"
	_ "embed"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/mgdunn2/cube-datahub/cubes/cards"
	"github.com/mgdunn2/cube-datahub/cubes/cubedb"
	"github.com/mgdunn2/cube-datahub/cubes/llm"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"log"
	"os"
)

func main() {
	ctx := context.Background()
	db := mustDb("local")
	storage := cubedb.NewStorage(db)
	openAiApiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(option.WithAPIKey(openAiApiKey))
	imageReader := llm.NewOpenAi(client)
	ccr := cards.NewLLMCustomCardReader(imageReader)
	cardLoader := cards.NewScryfallLoader(storage)
	cubeLoader := cards.NewCubeCobraLoader(storage, cardLoader, ccr)
	err := cubeLoader.Load(ctx, "da519447-9b91-4eac-a6d6-8a263f42e093")
	if err != nil {
		log.Fatal(fmt.Errorf(`load cube: %w`, err))
	}

	fmt.Println(`Loaded Cube!`)
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
