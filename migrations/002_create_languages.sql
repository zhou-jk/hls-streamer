-- Migration 002: Languages
CREATE TABLE IF NOT EXISTS `languages` (
    `code`        VARCHAR(10) NOT NULL,
    `name`        VARCHAR(100) NOT NULL,
    `native_name` VARCHAR(100) NOT NULL,
    `is_default`  TINYINT(1) NOT NULL DEFAULT 0,
    `is_active`   TINYINT(1) NOT NULL DEFAULT 1,
    `sort_order`  INT NOT NULL DEFAULT 0,
    PRIMARY KEY (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `languages` (`code`, `name`, `native_name`, `is_default`, `is_active`, `sort_order`) VALUES
('en',    'English',                'English',   1, 1, 1),
('zh-CN', 'Chinese (Simplified)',   '简体中文',   0, 1, 2),
('zh-TW', 'Chinese (Traditional)',  '繁體中文',   0, 1, 3),
('ja',    'Japanese',               '日本語',     0, 1, 4),
('ko',    'Korean',                 '한국어',     0, 1, 5);
