CREATE TABLE IF NOT EXISTS `reviews` (
  `userID` integer NOT NULL,
  `snippetID` integer NOT NULL,
  `review` tinyint UNSIGNED DEFAULT 0,
  PRIMARY KEY (`userID`, `snippetID`),
  CONSTRAINT `FK_snippet_reviews` FOREIGN KEY (`snippetID`) REFERENCES snippets(id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `FK_user_reviews` FOREIGN KEY (`userID`) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `reviews_max_review` CHECK (`review` between 0 and 5)
);
