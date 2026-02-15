-- Migration 007: Workers and Resolution Presets
CREATE TABLE IF NOT EXISTS `workers` (
    `id`              VARCHAR(100) NOT NULL,
    `hostname`        VARCHAR(255) NOT NULL,
    `ip_address`      VARCHAR(45) NOT NULL,
    `capabilities`    JSON NOT NULL,
    `status`          ENUM('online','busy','offline') NOT NULL DEFAULT 'offline',
    `current_task_id` INT UNSIGNED NULL,
    `last_heartbeat`  DATETIME NOT NULL,
    `registered_at`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    FOREIGN KEY (`current_task_id`) REFERENCES `transcode_tasks`(`id`) ON DELETE SET NULL,
    INDEX `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `resolution_presets` (
    `id`                 INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name`               VARCHAR(20) NOT NULL UNIQUE,
    `width`              INT UNSIGNED NOT NULL,
    `height`             INT UNSIGNED NOT NULL,
    `bitrate_kbps`       INT UNSIGNED NOT NULL,
    `audio_bitrate_kbps` INT UNSIGNED NOT NULL DEFAULT 128,
    `is_active`          TINYINT(1) NOT NULL DEFAULT 1,
    `sort_order`         INT NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `resolution_presets` (`name`, `width`, `height`, `bitrate_kbps`, `audio_bitrate_kbps`, `sort_order`) VALUES
('360p',  640,  360,  800,   96,  1),
('480p',  854,  480,  1200,  96,  2),
('720p',  1280, 720,  2500,  128, 3),
('1080p', 1920, 1080, 4500,  128, 4),
('1440p', 2560, 1440, 8000,  192, 5),
('2160p', 3840, 2160, 15000, 192, 6);
