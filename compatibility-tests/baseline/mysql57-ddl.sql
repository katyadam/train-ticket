-- Captured from the untouched Java baseline at commit
-- 313886e99befb94be6cd45f085c98e0019f59829 using MySQL 5.7.

CREATE TABLE `station` (
  `id` varchar(36) NOT NULL,
  `name` varchar(255) DEFAULT NULL,
  `stay_time` int(11) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `UK_gnneuc0peq2qi08yftdjhy7ok` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

CREATE TABLE `consign_record` (
  `consign_record_id` varchar(255) NOT NULL,
  `user_id` varchar(255) DEFAULT NULL,
  `consignee` varchar(255) DEFAULT NULL,
  `from_place` varchar(255) DEFAULT NULL,
  `handle_date` varchar(255) DEFAULT NULL,
  `order_id` varchar(255) DEFAULT NULL,
  `consign_record_phone` varchar(255) DEFAULT NULL,
  `consign_record_price` double DEFAULT NULL,
  `target_date` varchar(255) DEFAULT NULL,
  `to_place` varchar(255) DEFAULT NULL,
  `weight` double NOT NULL,
  PRIMARY KEY (`consign_record_id`)
) ENGINE=MyISAM DEFAULT CHARSET=latin1;
