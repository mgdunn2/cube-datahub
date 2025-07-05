package main

import (
	"context"
	_ "embed"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/mgdunn2/cube-datahub/cubes"
	"github.com/mgdunn2/cube-datahub/cubes/cards"
	"github.com/mgdunn2/cube-datahub/cubes/cubedb"
	"github.com/mgdunn2/cube-datahub/cubes/llm"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	ctx := context.Background()
	db := mustDb("local")
	s := cubedb.NewStorage(db)
	openAiApiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(option.WithAPIKey(openAiApiKey))
	imageReader := llm.NewOpenAi(client)
	dr := cards.NewLLMDeckReader(s, imageReader)
	httpClient := http.Client{}
	rsp, err := httpClient.Get(`https://media.discordapp.net/attachments/1372322448720007320/1382329190925467729/IMG_0831.jpg?ex=686a65e1&is=68691461&hm=d4abffbdb1ab5392bd1a66f729539fbc8a610a8a2801d246dc58b6b4dcdc90db&=&format=webp&width=1852&height=1390`)
	if err != nil {
		log.Panicln(fmt.Errorf(`get deck: %w`, err))
	}
	bytes, err := io.ReadAll(rsp.Body)
	if err != nil {
		log.Panicln(fmt.Errorf(`read body: %w`, err))
	}
	cube, err := s.GetCube(ctx, "da519447-9b91-4eac-a6d6-8a263f42e093", nil)
	if err != nil {
		log.Panicln(fmt.Errorf(`get cube: %w`, err))
	}
	e := cubes.Event{
		Cube: *cube,
	}
	d := cubes.Deck{
		Event: e,
	}
	d, err = dr.ReadDeck(ctx, d, bytes)
	if err != nil {
		log.Panicln(fmt.Errorf(`read deck: %w`, err))
	}

	for _, c := range d.Cards {
		fmt.Println(c.Name)
	}
	fmt.Println("Tada!")
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
