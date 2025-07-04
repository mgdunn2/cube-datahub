package cubes

import "context"

type Storage interface {
	// AddPlayer adds a new player
	AddPlayer(ctx context.Context, player Player) error

	// GetByNames returns a card by its name
	GetByNames(ctx context.Context, names []string) ([]Card, error)

	// GetByIDs returns cards by IDs
	GetByIDs(ctx context.Context, ids []string) ([]Card, error)

	// UpsertCards upserts a set of cards
	UpsertCards(ctx context.Context, cards []Card) error

	// AddCustomCard adds a mapping from an imageURL to the cardID for that card
	AddCustomCard(ctx context.Context, imageURL, cardID string) error

	// GetAllCustomCardIDs returns all custom card ID mappings ImageURL -> CardID
	GetAllCustomCardIDs(ctx context.Context) (map[string]string, error)

	// UpdateCube adds a new version of the cube
	UpdateCube(ctx context.Context, cube Cube) error

	// GetCube returns the cube at the specified version or the most recent if no version is provided
	GetCube(ctx context.Context, id string, version *int) (*Cube, error)

	// RecordDeck stores a deck
	RecordDeck(ctx context.Context, deck Deck) error
}
