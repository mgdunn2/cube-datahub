package main

import (
	"context"
	_ "embed"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/mgdunn2/cube-datahub/cubes/cards"
	"github.com/mgdunn2/cube-datahub/cubes/cubedb"
	"log"
)

func main() {
	ctx := context.Background()
	db := mustDb("local")
	storage := cubedb.NewStorage(db)
	cardLoader := cards.NewScryfallLoader(storage)

	cubeLoader := cards.NewCubeCobraLoader(storage, cardLoader)
	err := cubeLoader.Load(ctx, "b5e2e43a-a780-4b90-98bf-8097eaf7ff0f")
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
