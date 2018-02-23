CREATE TABLE IF NOT EXISTS `raceHistory` (
	`raceId` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
	`raceDatetime` DATETIME NOT NULL,
	PRIMARY KEY (`raceId`)
) COLLATE='utf8_general_ci' ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `raceHistoryStats` (
	`raceId` INT(11) UNSIGNED NOT NULL,
	`racerId` VARCHAR(255) NOT NULL,
	`racerUsername` VARCHAR(255) NOT NULL,
	`racerSpeed` FLOAT(7,6) NOT NULL,
	`racerTime` INT(11) NOT NULL,
	`place` INT(11) NOT NULL,
	PRIMARY KEY (`raceId`, `racerId`),
	CONSTRAINT `FK_raceHistoryStats_raceHistory` FOREIGN KEY (`raceId`) REFERENCES `raceHistory` (`raceId`)
) COLLATE='utf8_general_ci' ENGINE=InnoDB;