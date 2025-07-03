package cubes

import (
	"strconv"
	"strings"
	"time"
)

type Color string

const (
	White Color = "W"
	Blue  Color = "U"
	Black Color = "B"
	Red   Color = "R"
	Green Color = "G"
)

type Card struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ManaCost    *string   `json:"mana_cost,omitempty"`
	ManaValue   int       `json:"mana_value,omitempty"`
	Type        string    `json:"type"`
	SuperType   []string  `json:"super_type,omitempty"`
	SubType     []string  `json:"sub_type,omitempty"`
	Power       int       `json:"power,omitempty"`
	Toughness   int       `json:"toughness,omitempty"`
	Loyalty     int       `json:"loyalty,omitempty"`
	Defense     int       `json:"defense,omitempty"`
	Colors      []Color   `json:"colors,omitempty"`
	Set         string    `json:"set"`
	ReleaseDate time.Time `json:"release_date"`
}

type Cube struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	VersionNumber int       `json:"version_number"`
	Cards         []Card    `json:"cards"`
	Date          time.Time `json:"date"`
}

type Player struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Deck struct {
	ID     string `json:"id"`
	Player Player `json:"player"`
	Cube   Cube   `json:"cube"`
	Cards  []Card `json:"cards"`
}

// All third party models and conversions

// ScryfallCard is a trimmed-down model for just the fields we care about.
type ScryfallCard struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	ManaCost   *string  `json:"mana_cost"`
	Cmc        float64  `json:"cmc"`
	TypeLine   string   `json:"type_line"`
	Power      *string  `json:"power"`
	Toughness  *string  `json:"toughness"`
	Loyalty    *string  `json:"loyalty"`
	Defense    *string  `json:"defense"`
	Colors     []string `json:"colors"`
	Set        string   `json:"set"`
	ReleasedAt string   `json:"released_at"`
}

// ToCard converts a ScryfallCard into a domain-level Card model.
func (s ScryfallCard) ToCard() (Card, error) {
	superTypes, cardType, subTypes := parseTypeLine(s.TypeLine)

	card := Card{
		ID:        s.ID,
		Name:      s.Name,
		ManaCost:  s.ManaCost,
		ManaValue: int(s.Cmc),
		Type:      cardType,
		SuperType: superTypes,
		SubType:   subTypes,
		Set:       s.Set,
	}

	// Parse release date
	if t, err := time.Parse("2006-01-02", s.ReleasedAt); err == nil {
		card.ReleaseDate = t
	} else {
		return Card{}, err
	}

	// Parse power/toughness/loyalty/defense if numeric
	if s.Power != nil {
		if p, err := parseIntValue(*s.Power); err == nil {
			card.Power = p
		}
	}
	if s.Toughness != nil {
		if t, err := parseIntValue(*s.Toughness); err == nil {
			card.Toughness = t
		}
	}
	if s.Loyalty != nil {
		if l, err := parseIntValue(*s.Loyalty); err == nil {
			card.Loyalty = l
		}
	}
	if s.Defense != nil {
		if d, err := parseIntValue(*s.Defense); err == nil {
			card.Defense = d
		}
	}

	// Convert colors
	for _, c := range s.Colors {
		card.Colors = append(card.Colors, Color(c))
	}

	return card, nil
}

func parseTypeLine(typeLine string) (superTypes []string, cardType string, subTypes []string) {
	// Example typeLine: "Legendary Creature — Elf Warrior"
	parts := strings.Split(typeLine, "—")
	left := strings.Fields(strings.TrimSpace(parts[0]))

	var supertypes []string
	var cardtype string
	sawCardType := false
	for _, t := range left {
		if !sawCardType && isCardType(t) {
			cardtype = t
			sawCardType = true
		} else if sawCardType {
			cardtype += " " + t
		} else {
			supertypes = append(supertypes, t)
		}
	}

	var subtypes []string
	if len(parts) > 1 {
		subtypes = strings.Fields(strings.TrimSpace(parts[1]))
	}

	return supertypes, cardtype, subtypes
}

func isCardType(s string) bool {
	cardTypes := map[string]struct{}{
		"Artifact": {}, "Battle": {}, "Creature": {}, "Enchantment": {}, "Instant": {},
		"Land": {}, "Planeswalker": {}, "Sorcery": {},
	}
	_, ok := cardTypes[s]
	return ok
}

func parseIntValue(s string) (int, error) {
	return strconv.Atoi(s)
}

type CubeCobraCube struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Cards CubeCobraCards `json:"cards"`
}

type CubeCobraCards struct {
	MainBoard []CubeCobraCard `json:"mainboard"`
}

type CubeCobraCard struct {
	ID      string              `json:"cardID"`
	Details CubeCobraCardDetail `json:"details"`
}

type CubeCobraCardDetail struct {
	ScyfallID string `json:"scryfall_id	"`
}
