package cards

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mgdunn2/cube-datahub/cubes"
	"github.com/mgdunn2/cube-datahub/cubes/llm"
)

type DeckReader interface {
	ReadDeck(ctx context.Context, deck cubes.Deck, image []byte) (cubes.Deck, error)
}

type LLMDeckReader struct {
	s  cubes.Storage
	ir llm.ImageReader
}

func NewLLMDeckReader(s cubes.Storage, ir llm.ImageReader) *LLMDeckReader {
	return &LLMDeckReader{
		s:  s,
		ir: ir,
	}
}

func (l *LLMDeckReader) ReadDeck(ctx context.Context, deck cubes.Deck, image []byte) (cubes.Deck, error) {
	cardNamesStr := ""
	for i, card := range deck.Event.Cube.Cards {
		cardNamesStr += fmt.Sprintf("* %s", card.Name)
		if i != len(deck.Cards)-1 {
			cardNamesStr += "\n"
		}
	}
	req := llm.Request{
		Prompt: fmt.Sprintf(`Your job is to look at an image of a set of Magic cards that comprise a Vintage Cube deck and output
all of the cards that you see in the picture. Below is a list of every possible card that could be present. Some of them
are not real magic cards but all will have names that correspond to one of the cards in the provided list.

The image may have glare and may be rotate but do your best to identify every card that you can see the name of.

Every name that you return should exactly match one of the card names listed below.

You MUST return EVERY card that can be identified. Make sure to find a name for EVERY card in the image.

When you determine the cards that are found you should invoke the provided tool with the provided schema to return all
of the card names.

## Card List
%s
`, cardNamesStr),
		ImageByes: image,
		Schema: llm.ToolSchema{
			Name:        "deck",
			Description: "A deck of Magic: The Gathering cards names.",
			Schema:      llm.GenerateSchema(cubes.LLMDeckSchema{}),
		},
	}
	rsp, err := l.ir.Generate(ctx, req)
	if err != nil {
		return cubes.Deck{}, fmt.Errorf(`generate: %w`, err)
	}
	var llmDeck cubes.LLMDeckSchema
	if err = json.Unmarshal([]byte(rsp), &llmDeck); err != nil {
		return cubes.Deck{}, fmt.Errorf(`unmarshall: %w`, err)
	}
	cardsByName := make(map[string]cubes.Card)
	for _, card := range deck.Event.Cube.Cards {
		cardsByName[card.Name] = card
	}
	for _, cardName := range llmDeck.CardNames {
		if card, ok := cardsByName[cardName]; ok {
			deck.Cards = append(deck.Cards, card)
		}
	}
	return deck, nil
}
