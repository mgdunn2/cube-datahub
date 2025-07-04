package cubedb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mgdunn2/cube-datahub/cubes"
)

type storage struct {
	db *sqlx.DB
}

func NewStorage(db *sqlx.DB) cubes.Storage {
	return &storage{db: db}
}

type dbCard struct {
	ID          string         `db:"id"`
	Name        string         `db:"name"`
	ManaCost    sql.NullString `db:"mana_cost"`
	ManaValue   sql.NullInt64  `db:"mana_value"`
	Type        string         `db:"type"`
	SuperType   sql.NullString `db:"super_type"`
	SubType     sql.NullString `db:"sub_type"`
	TextBox     string         `db:"text_box"`
	Power       sql.NullInt64  `db:"power"`
	Toughness   sql.NullInt64  `db:"toughness"`
	Loyalty     sql.NullInt64  `db:"loyalty"`
	Defense     sql.NullInt64  `db:"defense"`
	Colors      sql.NullString `db:"colors"`
	Set         string         `db:"exp"`
	ReleaseDate time.Time      `db:"release_date"`
}

type dbPlayer struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}

type dbCube struct {
	ID         string `db:"id"`
	Name       string `db:"name"`
	MaxVersion int    `db:"maxVersion"`
}

type dbCubeVersion struct {
	CubeID        string    `db:"cubeId"`
	VersionNumber int       `db:"versionNumber"`
	Date          time.Time `db:"date"`
}

type dbCubeCard struct {
	CubeID        string `db:"cubeId"`
	VersionNumber int    `db:"versionNumber"`
	CardID        string `db:"cardId"`
}

type dbDeck struct {
	ID            string `db:"id"`
	PlayerID      string `db:"playerId"`
	CubeID        string `db:"cubeId"`
	VersionNumber int    `db:"versionNumber"`
	Description   string `db:"description"`
}

type dbDeckCard struct {
	DeckID string `db:"deckId"`
	CardID string `db:"cardId"`
}

// --- Conversion Helpers ---

func cardToDB(c cubes.Card) (*dbCard, error) {
	superType, _ := json.Marshal(c.SuperType)
	subType, _ := json.Marshal(c.SubType)
	colors, _ := json.Marshal(c.Colors)

	return &dbCard{
		ID:          c.ID,
		Name:        c.Name,
		ManaCost:    nullString(c.ManaCost),
		ManaValue:   nullInt(c.ManaValue),
		Type:        c.Type,
		SuperType:   nullJSONString(superType),
		SubType:     nullJSONString(subType),
		TextBox:     c.TextBox,
		Power:       nullInt(c.Power),
		Toughness:   nullInt(c.Toughness),
		Loyalty:     nullInt(c.Loyalty),
		Defense:     nullInt(c.Defense),
		Colors:      nullJSONString(colors),
		Set:         c.Set,
		ReleaseDate: c.ReleaseDate,
	}, nil
}

func dbToCard(c dbCard) (cubes.Card, error) {
	var superType, subType []string
	var colors []cubes.Color
	_ = json.Unmarshal([]byte(c.SuperType.String), &superType)
	_ = json.Unmarshal([]byte(c.SubType.String), &subType)
	_ = json.Unmarshal([]byte(c.Colors.String), &colors)

	var manaCost *string
	if c.ManaCost.Valid {
		manaCost = &c.ManaCost.String
	}

	return cubes.Card{
		ID:          c.ID,
		Name:        c.Name,
		ManaCost:    manaCost,
		ManaValue:   intOrZero(c.ManaValue),
		Type:        c.Type,
		SuperType:   superType,
		SubType:     subType,
		TextBox:     c.TextBox,
		Power:       intOrZero(c.Power),
		Toughness:   intOrZero(c.Toughness),
		Loyalty:     intOrZero(c.Loyalty),
		Defense:     intOrZero(c.Defense),
		Colors:      colors,
		Set:         c.Set,
		ReleaseDate: c.ReleaseDate,
	}, nil
}

func nullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: *s}
}

func nullJSONString(b []byte) sql.NullString {
	if len(b) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: string(b)}
}

func nullInt(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Valid: true, Int64: int64(i)}
}

func intOrZero(i sql.NullInt64) int {
	if i.Valid {
		return int(i.Int64)
	}
	return 0
}

// --- Storage Implementation ---

func (s *storage) AddPlayer(ctx context.Context, player cubes.Player) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO players (id, name) VALUES (?, ?)`,
		player.ID, player.Name,
	)
	return err
}

func (s *storage) GetByNames(ctx context.Context, names []string) ([]cubes.Card, error) {
	query, args, err := sqlx.In(`SELECT * FROM cards WHERE name IN (?)`, names)
	if err != nil {
		return nil, err
	}
	query = s.db.Rebind(query)
	var dbs []dbCard
	if err := s.db.SelectContext(ctx, &dbs, query, args...); err != nil {
		return nil, err
	}
	var cards []cubes.Card
	for _, dbCard := range dbs {
		card, _ := dbToCard(dbCard)
		cards = append(cards, card)
	}
	return cards, nil
}

func (s *storage) GetByIDs(ctx context.Context, ids []string) ([]cubes.Card, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := sqlx.In(`SELECT * FROM cards WHERE id IN (?)`, ids)
	if err != nil {
		return nil, err
	}
	query = s.db.Rebind(query)

	var dbs []dbCard
	if err := s.db.SelectContext(ctx, &dbs, query, args...); err != nil {
		return nil, err
	}

	cardMap := make(map[string]cubes.Card, len(dbs))
	for _, dc := range dbs {
		card, err := dbToCard(dc)
		if err != nil {
			return nil, fmt.Errorf("dbToCard: %w", err)
		}
		cardMap[card.ID] = card
	}

	cards := make([]cubes.Card, 0, len(ids))
	for _, id := range ids {
		if card, ok := cardMap[id]; ok {
			cards = append(cards, card)
		}
	}

	return cards, nil
}

func (s *storage) UpsertCards(ctx context.Context, cards []cubes.Card) error {
	const batchSize = 200
	if len(cards) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(`begin txn: %w`, err)
	}
	defer tx.Rollback()

	// Split into batches
	for i := 0; i < len(cards); i += batchSize {
		end := i + batchSize
		if end > len(cards) {
			end = len(cards)
		}
		if err := s.upsertCardBatch(ctx, tx, cards[i:end]); err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf(`commit txn: %w`, err)
	}
	return nil
}

func (s *storage) upsertCardBatch(ctx context.Context, tx *sql.Tx, cards []cubes.Card) error {
	if len(cards) == 0 {
		return nil
	}

	// Prepare value placeholders and args
	var (
		valueStrings []string
		args         []interface{}
	)

	for _, c := range cards {
		dbCard, err := cardToDB(c)
		if err != nil {
			return err
		}

		// Append placeholders for one row
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

		// Add all fields in order
		args = append(args,
			dbCard.ID,
			dbCard.Name,
			dbCard.ManaCost,
			dbCard.ManaValue,
			dbCard.Type,
			dbCard.SuperType,
			dbCard.SubType,
			dbCard.TextBox,
			dbCard.Power,
			dbCard.Toughness,
			dbCard.Loyalty,
			dbCard.Defense,
			dbCard.Colors,
			dbCard.Set,
			dbCard.ReleaseDate,
		)
	}

	stmt := `
INSERT INTO cards (
	id, name, mana_cost, mana_value, type, super_type, sub_type, text_box,
	power, toughness, loyalty, defense, colors, exp, release_date
) VALUES ` + strings.Join(valueStrings, ",") + `
ON DUPLICATE KEY UPDATE
	name=VALUES(name), mana_cost=VALUES(mana_cost), mana_value=VALUES(mana_value),
	type=VALUES(type), super_type=VALUES(super_type), sub_type=VALUES(sub_type), text_box=VALUES(text_box),
	power=VALUES(power), toughness=VALUES(toughness), loyalty=VALUES(loyalty),
	defense=VALUES(defense), colors=VALUES(colors), exp=VALUES(exp), release_date=VALUES(release_date)
`

	_, err := tx.ExecContext(ctx, stmt, args...)
	return err
}

func (s *storage) UpdateCube(ctx context.Context, cube cubes.Cube) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `INSERT INTO cubes (id, name, maxVersion)
		VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE name=VALUES(name), maxVersion=VALUES(maxVersion)`,
		cube.ID, cube.Name, cube.VersionNumber)
	if err != nil {
		return fmt.Errorf(`insert cube: %w`, err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO cube_versions (cubeId, versionNumber, date) VALUES (?, ?, ?)`,
		cube.ID, cube.VersionNumber, cube.Date)
	if err != nil {
		return fmt.Errorf(`insert cube version: %w`, err)
	}

	if len(cube.Cards) > 0 {
		counts := make(map[string]int)
		for _, card := range cube.Cards {
			counts[card.ID]++
		}

		valueStrings := make([]string, 0, len(counts))
		args := make([]interface{}, 0, len(counts)*4)
		for cardID, count := range counts {
			valueStrings = append(valueStrings, "(?, ?, ?, ?)")
			args = append(args, cube.ID, cube.VersionNumber, cardID, count)
		}

		stmt := `INSERT INTO cube_cards (cubeId, versionNumber, cardId, count) VALUES ` + strings.Join(valueStrings, ",")
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return fmt.Errorf(`insert cube cards batch: %w`, err)
		}
	}
	return tx.Commit()
}

func (s *storage) GetCube(ctx context.Context, id string, version *int) (*cubes.Cube, error) {
	var v int
	if version == nil {
		err := s.db.GetContext(ctx, &v, `SELECT maxVersion FROM cubes WHERE id = ?`, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, fmt.Errorf(`get max version: %w`, err)
		}
		version = &v
	}

	var cv dbCubeVersion
	err := s.db.GetContext(ctx, &cv, `SELECT * FROM cube_versions WHERE cubeId = ? AND versionNumber = ?`, id, *version)
	if err != nil {
		return nil, fmt.Errorf(`get cube version: %w`, err)
	}

	var cube dbCube
	err = s.db.GetContext(ctx, &cube, `SELECT * FROM cubes WHERE id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf(`get cube: %w`, err)
	}

	var cardIDs []string
	err = s.db.SelectContext(ctx, &cardIDs, `SELECT cardId FROM cube_cards WHERE cubeId = ? AND versionNumber = ?`, id, *version)
	if err != nil {
		return nil, fmt.Errorf(`get cube card IDs: %w`, err)
	}

	cards, err := s.GetByIDs(ctx, cardIDs)
	if err != nil {
		return nil, fmt.Errorf(`get cards: %w`, err)
	}

	return &cubes.Cube{
		ID:            id,
		Name:          cube.Name,
		VersionNumber: *version,
		Date:          cv.Date,
		Cards:         cards,
	}, nil
}

func (s *storage) RecordDeck(ctx context.Context, deck cubes.Deck) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO decks (id, playerId, cubeId, versionNumber, description) VALUES (?, ?, ?, ?, ?)`,
		deck.ID, deck.Player.ID, deck.Cube.ID, deck.Cube.VersionNumber, deck.Player.Name)
	if err != nil {
		return err
	}

	for _, card := range deck.Cards {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO deck_cards (deckId, cardId) VALUES (?, ?)`,
			deck.ID, card.ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
