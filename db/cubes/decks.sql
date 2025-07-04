CREATE TABLE decks (
  `id` CHAR(36) PRIMARY KEY,
  `playerId` CHAR(36) NOT NULL,
  `eventId` CHAR(36) NOT NULL,
  `description` varchar(255) NOT NULL,
  `imageUrl` varchar(255) NOT NULL
);
