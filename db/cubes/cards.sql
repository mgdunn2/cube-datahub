CREATE TABLE cards (
  `id` CHAR(36) PRIMARY KEY,       
  `name` VARCHAR(255) NOT NULL,
  `mana_cost` VARCHAR(255), 
  `mana_value` int,
  `type` VARCHAR(64) NOT NULL,
  `super_type` JSON,
  `sub_type` JSON,
  `power` int,
  `toughness` int,
  `loyalty` int,
  `defense` int,
  `colors` JSON,
  `exp` VARCHAR(31) NOT NULL,
  `release_date` DATE NOT NULL,
  KEY `name` (`name`)
);
