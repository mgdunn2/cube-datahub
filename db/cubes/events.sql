CREATE TABLE events (
  `id` CHAR(36) PRIMARY KEY,
  `cubeId` CHAR(36) NOT NULL,
  `versionNumber` int NOT NULL,
  `eventDate` TIMESTAMP not null
);
