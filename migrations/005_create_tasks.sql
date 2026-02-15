-- Migration 005: Transcode Tasks
CREATE TABLE IF NOT EXISTS `transcode_tasks` (
    `id`            INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `task_uuid`     CHAR(36) NOT NULL UNIQUE,
    `video_id`      INT UNSIGNED NOT NULL,
    `type`          ENUM('transcode','thumbnail','drm_package','probe') NOT NULL,
    `status`        ENUM('pending','queued','processing','completed','failed','cancelled') NOT NULL DEFAULT 'pending',
    `priority`      INT NOT NULL DEFAULT 0,
    `worker_id`     VARCHAR(100) NULL,
    `params`        JSON NOT NULL,
    `result`        JSON NULL,
    `error_message` TEXT NULL,
    `progress`      TINYINT UNSIGNED NOT NULL DEFAULT 0,
    `attempts`      INT UNSIGNED NOT NULL DEFAULT 0,
    `max_attempts`  INT UNSIGNED NOT NULL DEFAULT 3,
    `started_at`    DATETIME NULL,
    `completed_at`  DATETIME NULL,
    `created_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    FOREIGN KEY (`video_id`) REFERENCES `videos`(`id`) ON DELETE CASCADE,
    INDEX `idx_status_priority` (`status`, `priority` DESC),
    INDEX `idx_worker` (`worker_id`),
    INDEX `idx_video_type` (`video_id`, `type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `task_logs` (
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `task_id`    INT UNSIGNED NOT NULL,
    `level`      ENUM('info','warn','error') NOT NULL DEFAULT 'info',
    `message`    TEXT NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    FOREIGN KEY (`task_id`) REFERENCES `transcode_tasks`(`id`) ON DELETE CASCADE,
    INDEX `idx_task` (`task_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
