-- Migration 003: Videos and related tables
CREATE TABLE IF NOT EXISTS `videos` (
    `id`                  INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `uuid`                CHAR(36) NOT NULL UNIQUE,
    `slug`                VARCHAR(255) NOT NULL UNIQUE,
    `status`              ENUM('draft','processing','ready','error','archived') NOT NULL DEFAULT 'draft',
    `original_filename`   VARCHAR(500) NOT NULL DEFAULT '',
    `original_s3_key`     VARCHAR(1000) NOT NULL DEFAULT '',
    `duration_seconds`    DECIMAL(10,2) NULL,
    `width`               INT UNSIGNED NULL,
    `height`              INT UNSIGNED NULL,
    `file_size_bytes`     BIGINT UNSIGNED NULL,
    `codec`               VARCHAR(50) NULL,
    `fps`                 DECIMAL(6,2) NULL,
    `has_drm`             TINYINT(1) NOT NULL DEFAULT 0,
    `master_playlist_key` VARCHAR(1000) NULL,
    `release_date`        DATE NULL,
    `rating`              VARCHAR(10) NULL,
    `sort_order`          INT NOT NULL DEFAULT 0,
    `view_count`          BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `created_by`          INT UNSIGNED NOT NULL,
    `created_at`          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`          DATETIME NULL,
    PRIMARY KEY (`id`),
    FOREIGN KEY (`created_by`) REFERENCES `users`(`id`),
    INDEX `idx_uuid` (`uuid`),
    INDEX `idx_status` (`status`),
    INDEX `idx_deleted` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `video_translations` (
    `id`              INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `video_id`        INT UNSIGNED NOT NULL,
    `language_code`   VARCHAR(10) NOT NULL,
    `title`           VARCHAR(500) NOT NULL,
    `description`     TEXT NULL,
    `synopsis`        TEXT NULL,
    `seo_title`       VARCHAR(200) NULL,
    `seo_description` VARCHAR(500) NULL,
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_video_lang` (`video_id`, `language_code`),
    FOREIGN KEY (`video_id`) REFERENCES `videos`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`language_code`) REFERENCES `languages`(`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `video_variants` (
    `id`               INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `video_id`         INT UNSIGNED NOT NULL,
    `resolution_name`  VARCHAR(20) NOT NULL,
    `width`            INT UNSIGNED NOT NULL,
    `height`           INT UNSIGNED NOT NULL,
    `bitrate_kbps`     INT UNSIGNED NOT NULL,
    `codec`            VARCHAR(50) NOT NULL DEFAULT 'h264',
    `playlist_s3_key`  VARCHAR(1000) NOT NULL DEFAULT '',
    `segment_count`    INT UNSIGNED NOT NULL DEFAULT 0,
    `total_size_bytes` BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `status`           ENUM('pending','processing','ready','error') NOT NULL DEFAULT 'pending',
    `created_at`       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_video_resolution` (`video_id`, `resolution_name`, `codec`),
    FOREIGN KEY (`video_id`) REFERENCES `videos`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `thumbnails` (
    `id`           INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `video_id`     INT UNSIGNED NOT NULL,
    `s3_key`       VARCHAR(1000) NOT NULL,
    `width`        INT UNSIGNED NOT NULL,
    `height`       INT UNSIGNED NOT NULL,
    `size_bytes`   INT UNSIGNED NOT NULL DEFAULT 0,
    `timestamp_s`  DECIMAL(10,2) NOT NULL DEFAULT 0,
    `is_default`   TINYINT(1) NOT NULL DEFAULT 0,
    `sort_order`   INT NOT NULL DEFAULT 0,
    `created_at`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    FOREIGN KEY (`video_id`) REFERENCES `videos`(`id`) ON DELETE CASCADE,
    INDEX `idx_video_default` (`video_id`, `is_default`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `subtitles` (
    `id`            INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `video_id`      INT UNSIGNED NOT NULL,
    `language_code` VARCHAR(10) NOT NULL,
    `label`         VARCHAR(100) NOT NULL,
    `s3_key`        VARCHAR(1000) NOT NULL,
    `is_default`    TINYINT(1) NOT NULL DEFAULT 0,
    `sort_order`    INT NOT NULL DEFAULT 0,
    `created_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_video_subtitle_lang` (`video_id`, `language_code`),
    FOREIGN KEY (`video_id`) REFERENCES `videos`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`language_code`) REFERENCES `languages`(`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- People (cast/crew)
CREATE TABLE IF NOT EXISTS `people` (
    `id`         INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `slug`       VARCHAR(255) NOT NULL UNIQUE,
    `photo_url`  VARCHAR(1000) NULL,
    `birth_date` DATE NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `people_translations` (
    `id`            INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `person_id`     INT UNSIGNED NOT NULL,
    `language_code` VARCHAR(10) NOT NULL,
    `name`          VARCHAR(255) NOT NULL,
    `biography`     TEXT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_person_lang` (`person_id`, `language_code`),
    FOREIGN KEY (`person_id`) REFERENCES `people`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`language_code`) REFERENCES `languages`(`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `video_casts` (
    `id`             INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `video_id`       INT UNSIGNED NOT NULL,
    `person_id`      INT UNSIGNED NOT NULL,
    `role`           ENUM('director','actor','writer','producer','other') NOT NULL,
    `character_name` VARCHAR(255) NULL,
    `sort_order`     INT NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    FOREIGN KEY (`video_id`) REFERENCES `videos`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`person_id`) REFERENCES `people`(`id`) ON DELETE CASCADE,
    INDEX `idx_video_role` (`video_id`, `role`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
