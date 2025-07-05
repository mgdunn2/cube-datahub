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
	ManaCost    *string   `json:"mana_cost"`
	ManaValue   int       `json:"mana_value"`
	Type        string    `json:"type"`
	SuperType   []string  `json:"super_type"`
	SubType     []string  `json:"sub_type"`
	TextBox     string    `json:"text_box"`
	Power       int       `json:"power"`
	Toughness   int       `json:"toughness"`
	Loyalty     int       `json:"loyalty"`
	Defense     int       `json:"defense"`
	Colors      []Color   `json:"colors"`
	Set         string    `json:"set"`
	ReleaseDate time.Time `json:"release_date"`
	ImageURI    string    `json:"image_uri"`
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

type Event struct {
	ID   string    `json:"id"`
	Cube Cube      `json:"cube"`
	Date time.Time `json:"date"`
}

type Deck struct {
	ID          string `json:"id"`
	PlayerID    string `json:"playerId"`
	EventID     string `json:"eventId"`
	Cards       []Card `json:"cards"`
	Description string `json:"description"`
}

// All third party models and conversions

type ScryfallCard struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	ManaCost   *string            `json:"mana_cost"`
	Cmc        float64            `json:"cmc"`
	TypeLine   string             `json:"type_line"`
	OracleText string             `json:"oracle_text"`
	Power      *string            `json:"power"`
	Toughness  *string            `json:"toughness"`
	Loyalty    *string            `json:"loyalty"`
	Defense    *string            `json:"defense"`
	Colors     []string           `json:"colors"`
	Set        string             `json:"set"`
	ReleasedAt string             `json:"released_at"`
	ImageURIs  *ScryfallImageURIs `json:"image_uris"`
	CardFaces  []ScryfallCardFace `json:"card_faces"`
}

type ScryfallCardFace struct {
	Name       string             `json:"name"`
	ManaCost   *string            `json:"mana_cost"`
	TypeLine   string             `json:"type_line"`
	OracleText string             `json:"oracle_text"`
	Power      *string            `json:"power"`
	Toughness  *string            `json:"toughness"`
	Loyalty    *string            `json:"loyalty"`
	Defense    *string            `json:"defense"`
	Colors     []string           `json:"colors"`
	ImageURIs  *ScryfallImageURIs `json:"image_uris"`
}

type ScryfallImageURIs struct {
	Small  string `json:"small"`
	Normal string `json:"normal"`
	Large  string `json:"large"`
}

func (s ScryfallCard) ToCard() (Card, error) {
	var (
		name       = s.Name
		manaCost   = s.ManaCost
		typeLine   = s.TypeLine
		oracleText = s.OracleText
		power      = s.Power
		toughness  = s.Toughness
		loyalty    = s.Loyalty
		defense    = s.Defense
		colors     = s.Colors
		imageURIs  = s.ImageURIs
	)

	if len(s.CardFaces) > 0 {
		face := s.CardFaces[0]
		name = face.Name
		manaCost = face.ManaCost
		typeLine = face.TypeLine
		oracleText = face.OracleText
		power = face.Power
		toughness = face.Toughness
		loyalty = face.Loyalty
		defense = face.Defense
		colors = face.Colors
		if face.ImageURIs != nil {
			imageURIs = face.ImageURIs
		}
	}

	superTypes, cardType, subTypes := parseTypeLine(typeLine)

	card := Card{
		ID:        s.ID,
		Name:      name,
		ManaCost:  manaCost,
		ManaValue: int(s.Cmc),
		Type:      cardType,
		SuperType: superTypes,
		SubType:   subTypes,
		TextBox:   oracleText,
		Set:       s.Set,
	}

	if imageURIs != nil {
		card.ImageURI = imageURIs.Normal
	}

	if t, err := time.Parse("2006-01-02", s.ReleasedAt); err == nil {
		card.ReleaseDate = t
	} else {
		return Card{}, err
	}

	if power != nil {
		if p, err := parseIntValue(*power); err == nil {
			card.Power = p
		}
	}
	if toughness != nil {
		if t, err := parseIntValue(*toughness); err == nil {
			card.Toughness = t
		}
	}
	if loyalty != nil {
		if l, err := parseIntValue(*loyalty); err == nil {
			card.Loyalty = l
		}
	}
	if defense != nil {
		if d, err := parseIntValue(*defense); err == nil {
			card.Defense = d
		}
	}

	for _, c := range colors {
		card.Colors = append(card.Colors, Color(c))
	}

	return card, nil
}

// LLMCardSchema exists purely for being converted into an OpenAI request json schema
type LLMCardSchema struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	ManaCost   *string  `json:"mana_cost"`
	Cmc        float64  `json:"cmc"`
	TypeLine   string   `json:"type_line"`
	OracleText string   `json:"oracle_text"`
	Power      int      `json:"power"`
	Toughness  int      `json:"toughness"`
	Loyalty    int      `json:"loyalty"`
	Defense    int      `json:"defense"`
	Colors     []string `json:"colors"`
	Set        string   `json:"set"`
	ReleasedAt string   `json:"released_at"`
}

// ToCard converts a ScryfallCard into a domain-level Card model.
func (s LLMCardSchema) ToCard() (Card, error) {
	superTypes, cardType, subTypes := parseTypeLine(s.TypeLine)

	card := Card{
		ID:        s.ID,
		Name:      s.Name,
		ManaCost:  s.ManaCost,
		ManaValue: int(s.Cmc),
		Type:      cardType,
		SuperType: superTypes,
		SubType:   subTypes,
		TextBox:   s.OracleText,
		Power:     s.Power,
		Toughness: s.Toughness,
		Loyalty:   s.Loyalty,
		Defense:   s.Defense,
		Set:       s.Set,
	}

	// Parse release date
	if t, err := time.Parse("2006-01-02", s.ReleasedAt); err == nil {
		card.ReleaseDate = t
	} else {
		// Especially for custom cards, don't bother trying to get a correct date
		card.ReleaseDate = time.Now()
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
	ID       string              `json:"cardID"`
	Details  CubeCobraCardDetail `json:"details"`
	Tags     []string            `json:"tags"`
	ImageURL string              `json:"imgUrl"`
}

type CubeCobraCardDetail struct {
	ScyfallID string `json:"scryfall_id	"`
}
