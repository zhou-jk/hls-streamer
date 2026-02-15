-- Migration 006: DRM Keys
CREATE TABLE IF NOT EXISTS `drm_keys` (
    `id`           INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `video_id`     INT UNSIGNED NOT NULL,
    `key_id`       VARCHAR(64) NOT NULL,
    `content_key`  VARCHAR(64) NOT NULL,
    `iv`           VARCHAR(64) NULL,
    `drm_system`   ENUM('widevine','fairplay','common') NOT NULL,
    `license_url`  VARCHAR(1000) NULL,
    `pssh_box`     TEXT NULL,
    `fairplay_uri` VARCHAR(1000) NULL,
    `created_at`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    FOREIGN KEY (`video_id`) REFERENCES `videos`(`id`) ON DELETE CASCADE,
    INDEX `idx_video_drm` (`video_id`, `drm_system`),
    UNIQUE KEY `uk_key_id` (`key_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
