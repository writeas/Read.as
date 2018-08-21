--
-- Table structure for table `follows`
--

CREATE TABLE IF NOT EXISTS `follows` (
  `follower` int(11) NOT NULL,
  `followee` int(11) NOT NULL,
  `created` datetime NOT NULL,
  PRIMARY KEY (`follower`,`followee`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

--
-- Table structure for table `localusers`
--

CREATE TABLE IF NOT EXISTS `localusers` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `username` varchar(60) NOT NULL,
  `password` char(60) NOT NULL,
  `name` varchar(100) NOT NULL,
  `summary` varchar(255) NOT NULL,
  `private_key` blob NOT NULL,
  `public_key` blob NOT NULL,
  `created` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

--
-- Table structure for table `posts`
--

CREATE TABLE IF NOT EXISTS `posts` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `owner_id` int(11) NOT NULL,
  `activity_id` varchar(255) NOT NULL,
  `type` varchar(10) NOT NULL,
  `published` datetime NOT NULL,
  `url` varchar(255) NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `content` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin,
  PRIMARY KEY (`id`),
  UNIQUE KEY `activity_id` (`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

--
-- Table structure for table `userkeys`
--

CREATE TABLE IF NOT EXISTS `userkeys` (
  `id` varchar(255) NOT NULL,
  `user_id` int(11) NOT NULL,
  `public_key` blob NOT NULL,
  `private_key` blob,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

--
-- Table structure for table `users`
--

CREATE TABLE IF NOT EXISTS `users` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `actor_id` varchar(255) NOT NULL,
  `username` varchar(60) NOT NULL,
  `type` varchar(10) NOT NULL,
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `summary` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `url` varchar(255) NOT NULL,
  `following_iri` varchar(255) NOT NULL,
  `followers_iri` varchar(255) NOT NULL,
  `inbox_iri` varchar(255) NOT NULL,
  `outbox_iri` varchar(255) NOT NULL,
  `shared_inbox_iri` varchar(255) NOT NULL,
  `avatar` varchar(255) NOT NULL,
  `avatar_type` varchar(255) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `actor_id` (`actor_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

