CREATE TABLE deck_cards (
  `deckId` CHAR(36) NOT NULL,
  `cardId` CHAR(36) NOT NULL,
  PRIMARY KEY (`deckId`, `cardId`)
);
