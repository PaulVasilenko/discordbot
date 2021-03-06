START TRANSACTION;

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

CREATE DATABASE IF NOT EXISTS pandabot;

USE pandabot;

CREATE TABLE IF NOT EXISTS `smileyHistory` (
  `emojiId` VARCHAR(20),
  `emojiName` VARCHAR(20) COLLATE latin1_general_cs,
  `userId` VARCHAR(20),
  `userName` VARCHAR(20),
  `createDatetime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `updateDatetime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (emojiId, userId, createDatetime)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='This table represents history of smiley usages.';

CREATE TABLE IF NOT EXISTS `raceHistory` (
	`raceId` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
	`raceDatetime` DATETIME NOT NULL,
	PRIMARY KEY (`raceId`)
) COLLATE='utf8_general_ci' ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `raceHistoryStats` (
	`raceId` INT(11) UNSIGNED NOT NULL,
	`racerId` VARCHAR(11) NOT NULL,
	`racerUsername` VARCHAR(11) NOT NULL,
	`racerSpeed` VARCHAR(11) NOT NULL,
	`racerTime` VARCHAR(11) NOT NULL,
	PRIMARY KEY (`raceId`, `racerId`),
	CONSTRAINT `FK_raceHistoryStats_raceHistory` FOREIGN KEY (`raceId`) REFERENCES `raceHistory` (`raceId`)
) COLLATE='utf8_general_ci' ENGINE=InnoDB;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

COMMIT;