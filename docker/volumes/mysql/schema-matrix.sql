-- docker/volumes/mysql/schema-matrix.sql
SET NAMES utf8mb4;
CREATE DATABASE IF NOT EXISTS sa_go COLLATE utf8mb4_unicode_ci;
USE sa_go;
CREATE TABLE IF NOT EXISTS `api_key` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `api_key` varchar(255) NOT NULL DEFAULT '',
  `name` varchar(255) NOT NULL DEFAULT '',
  `status` tinyint NOT NULL DEFAULT 0,
  `user_id` bigint NOT NULL DEFAULT 0,
  `expired_at` bigint NOT NULL DEFAULT 0,
  `created_at` bigint unsigned NOT NULL DEFAULT 0,
  `updated_at` bigint unsigned NOT NULL DEFAULT 0,
  `last_used_at` bigint NOT NULL DEFAULT 0,
  `ak_type` tinyint NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
INSERT IGNORE INTO `api_key` (`id`,`api_key`,`name`,`status`,`user_id`,`expired_at`,`created_at`,`updated_at`,`last_used_at`,`ak_type`)
VALUES (1,'matrix-admin-key','Matrix Admin Key',0,1,0,0,0,0,0);
