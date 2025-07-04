CREATE TABLE cube_cards (
  `cubeId` CHAR(36) NOT NULL,
  `versionNumber` int NOT NULL,
  `cardId` char(36) not null,
  `count` int not null,
  PRIMARY KEY (`cubeId`, `versionNumber`, `cardId`)
);
