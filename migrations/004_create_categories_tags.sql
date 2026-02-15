-- Migration 004: Categories and Tags
CREATE TABLE IF NOT EXISTS `categories` (
    `id`         INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `slug`       VARCHAR(255) NOT NULL UNIQUE,
    `parent_id`  INT UNSIGNED NULL,
    `sort_order` INT NOT NULL DEFAULT 0,
    `is_active`  TINYINT(1) NOT NULL DEFAULT 1,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    FOREIGN KEY (`parent_id`) REFERENCES `categories`(`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `category_translations` (
    `id`            INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `category_id`   INT UNSIGNED NOT NULL,
    `language_code` VARCHAR(10) NOT NULL,
    `name`          VARCHAR(255) NOT NULL,
    `description`   TEXT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_category_lang` (`category_id`, `language_code`),
    FOREIGN KEY (`category_id`) REFERENCES `categories`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`language_code`) REFERENCES `languages`(`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `tags` (
    `id`         INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `slug`       VARCHAR(255) NOT NULL UNIQUE,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `tag_translations` (
    `id`            INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tag_id`        INT UNSIGNED NOT NULL,
    `language_code` VARCHAR(10) NOT NULL,
    `name`          VARCHAR(255) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_tag_lang` (`tag_id`, `language_code`),
    FOREIGN KEY (`tag_id`) REFERENCES `tags`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`language_code`) REFERENCES `languages`(`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `video_categories` (
    `video_id`    INT UNSIGNED NOT NULL,
    `category_id` INT UNSIGNED NOT NULL,
    PRIMARY KEY (`video_id`, `category_id`),
    FOREIGN KEY (`video_id`) REFERENCES `videos`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`category_id`) REFERENCES `categories`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `video_tags` (
    `video_id` INT UNSIGNED NOT NULL,
    `tag_id`   INT UNSIGNED NOT NULL,
    PRIMARY KEY (`video_id`, `tag_id`),
    FOREIGN KEY (`video_id`) REFERENCES `videos`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`tag_id`) REFERENCES `tags`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
