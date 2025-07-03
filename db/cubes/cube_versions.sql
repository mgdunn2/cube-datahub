CREATE TABLE cube_versions (
  `cubeId` CHAR(36) NOT NULL,
  `versionNumber` int NOT NULL,
  `date` timestamp not null,
  PRIMARY KEY (`cubeId`, `versionNumber`)
);
