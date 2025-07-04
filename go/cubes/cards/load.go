package cards

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mgdunn2/cube-datahub/cubes"
	"io"
	"log"
	"net/http"
	"slices"
	"time"
)

type CardLoader interface {
	LoadCards(ctx context.Context, ids []string) error
}

type ScryfallApiCardLoader struct {
	client  *http.Client
	storage cubes.Storage
}

type ScryfallApiCardLoaderOpts func(*ScryfallApiCardLoader)

func ScryfallLoaderWithClient(client http.Client) ScryfallApiCardLoaderOpts {
	return func(c *ScryfallApiCardLoader) {
		c.client = &client
	}
}

func NewScryfallLoader(storage cubes.Storage, opts ...ScryfallApiCardLoaderOpts) *ScryfallApiCardLoader {
	loader := &ScryfallApiCardLoader{
		storage: storage,
	}
	for _, opt := range opts {
		opt(loader)
	}
	if loader.client == nil {
		loader.client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	return loader
}

type CollectionRequest struct {
	Identifiers []CardIdentifier `json:"identifiers"`
}
type CardIdentifier struct {
	ID string `json:"id"`
}

type CollectionResponse struct {
	Cards []cubes.ScryfallCard `json:"data"`
}

func (f *ScryfallApiCardLoader) LoadCards(ctx context.Context, ids []string) error {
	const batchSize = 75
	var allCards []cubes.Card

	var missingCards []string

	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		// Prepare batch
		cardIDs := make([]CardIdentifier, end-start)
		for i := start; i < end; i++ {
			cardIDs[i-start] = CardIdentifier{ID: ids[i]}
		}
		collectionRequest := CollectionRequest{
			Identifiers: cardIDs,
		}
		jsonReq, err := json.Marshal(collectionRequest)
		if err != nil {
			return fmt.Errorf(`marshal collection request: %w`, err)
		}

		rsp, err := f.client.Post("https://api.scryfall.com/cards/collection", "application/json", bytes.NewBuffer(jsonReq))
		if err != nil {
			return fmt.Errorf(`post collection: %w`, err)
		}
		bodyBytes, err := io.ReadAll(rsp.Body)
		_ = rsp.Body.Close()

		var response CollectionResponse
		if err := json.Unmarshal(bodyBytes, &response); err != nil {
			return fmt.Errorf(`unmarshal collection response: %w`, err)
		}

		foundCardIDs := make(map[string]struct{}, len(response.Cards))
		for _, scryfallCard := range response.Cards {
			card, err := scryfallCard.ToCard()
			if err != nil {
				log.Println(fmt.Errorf(`converting to card: %w`, err))
			}
			foundCardIDs[card.ID] = struct{}{}
			allCards = append(allCards, card)
		}
		for _, card := range cardIDs {
			if _, ok := foundCardIDs[card.ID]; !ok {
				missingCards = append(missingCards, card.ID)
			}
		}
	}
	if len(missingCards) > 0 {
		fmt.Printf(`Missing cards: %v\n`, missingCards)
	}

	// Final upsert
	if err := f.storage.UpsertCards(ctx, allCards); err != nil {
		return fmt.Errorf(`upsert cards: %w`, err)
	}
	return nil
}

type CubeLoader interface {
	Load(ctx context.Context, cubeID string) error
}

type CubeCobraLoader struct {
	client           *http.Client
	storage          cubes.Storage
	cardLoader       CardLoader
	customCardReader CustomCardReader
}

type CubeCobraLoaderOpts func(*CubeCobraLoader)

func CubeLoaderWithClient(client http.Client) CubeCobraLoaderOpts {
	return func(c *CubeCobraLoader) {
		c.client = &client
	}
}

func NewCubeCobraLoader(storage cubes.Storage, cardLoader CardLoader, customCardReader CustomCardReader, opts ...CubeCobraLoaderOpts) *CubeCobraLoader {
	loader := &CubeCobraLoader{
		storage:          storage,
		cardLoader:       cardLoader,
		customCardReader: customCardReader,
	}
	for _, opt := range opts {
		opt(loader)
	}
	if loader.client == nil {
		loader.client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	return loader
}

func (c *CubeCobraLoader) Load(ctx context.Context, cubeID string) error {
	url := fmt.Sprintf(`https://cubecobra.com/cube/api/cubeJSON/%s`, cubeID)

	rsp, err := c.client.Get(url)
	if err != nil {
		return fmt.Errorf(`get cube cobra: %w`, err)
	}
	bodyBytes, err := io.ReadAll(rsp.Body)
	if err != nil {
		return fmt.Errorf(`read bytes: %w`, err)
	}
	var cubeCobraCube cubes.CubeCobraCube
	if err := json.Unmarshal(bodyBytes, &cubeCobraCube); err != nil {
		return fmt.Errorf(`unmarshal: %w`, err)
	}

	customMappings, err := c.storage.GetAllCustomCardIDs(ctx)
	if err != nil {
		return fmt.Errorf(`getting custom mappings: %w`, err)
	}

	var cardIDs []string
	var customCardIDs []string
	for _, card := range cubeCobraCube.Cards.MainBoard {
		if slices.Index(card.Tags, "custom") != -1 {
			customCard, err := c.handleCustom(ctx, card.ImageURL, customMappings)
			if err != nil {
				return fmt.Errorf(`handle custom: %w`, err)
			}
			customCardIDs = append(customCardIDs, customCard.ID)
		} else {
			cardIDs = append(cardIDs, card.Details.ScyfallID)
		}
	}
	err = c.cardLoader.LoadCards(ctx, cardIDs)
	if err != nil {
		return fmt.Errorf(`load cards: %w`, err)
	}
	var allCardIDs []string
	allCardIDs = append(cardIDs, customCardIDs...)
	cards, err := c.storage.GetByIDs(ctx, allCardIDs)

	currentCube, err := c.storage.GetCube(ctx, cubeID, nil)
	if err != nil {
		return fmt.Errorf(`get cube: %w`, err)
	}
	if currentCube != nil && !hasChanges(cards, currentCube.Cards) {
		return nil
	}
	var version int
	if currentCube != nil {
		version = currentCube.VersionNumber + 1
	}
	newCube := cubes.Cube{
		ID:            cubeID,
		Name:          cubeCobraCube.Name,
		VersionNumber: version,
		Cards:         cards,
		Date:          time.Now(),
	}
	err = c.storage.UpdateCube(ctx, newCube)
	if err != nil {
		return fmt.Errorf(`update cube: %w`, err)
	}
	return nil
}

func (c *CubeCobraLoader) handleCustom(ctx context.Context, imageURL string, customMappings map[string]string) (cubes.Card, error) {
	if cardID, ok := customMappings[imageURL]; ok {
		cards, err := c.storage.GetByIDs(ctx, []string{cardID})
		if err != nil {
			return cubes.Card{}, fmt.Errorf(`get by id: %w`, err)
		}
		if len(cards) != 1 {
			return cubes.Card{}, errors.New(`custom card with mapping not found`)
		}
		return cards[0], nil
	}
	card, err := c.customCardReader.ReadCard(ctx, imageURL)
	if err != nil {
		return cubes.Card{}, fmt.Errorf(`read card: %w`, err)
	}
	cardID := "uuid.New()" // Placeholder for importing google uuid
	card.ID = cardID
	err = c.storage.AddCustomCard(ctx, imageURL, cardID)
	if err != nil {
		return cubes.Card{}, fmt.Errorf(`add custom card: %w`, err)
	}
	err = c.storage.UpsertCards(ctx, []cubes.Card{card})
	if err != nil {
		return cubes.Card{}, fmt.Errorf(`upsert card: %w`, err)
	}

	return card, nil
}

func hasChanges(a, b []cubes.Card) bool {
	if len(a) != len(b) {
		return true
	}
	aCards := make(map[string]struct{}, len(a))
	for _, card := range a {
		aCards[card.ID] = struct{}{}
	}
	for _, card := range b {
		if _, ok := aCards[card.ID]; !ok {
			return true
		}
	}
	bCards := make(map[string]struct{}, len(b))
	for _, card := range b {
		bCards[card.ID] = struct{}{}
	}
	for _, card := range a {
		if _, ok := bCards[card.ID]; !ok {
			return true
		}
	}
	return false
}
