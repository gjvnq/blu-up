BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS `volumes` (
	`uuid`	TEXT NOT NULL,
	`name`	TEXT NOT NULL,
	`desc`	TEXT NOT NULL,
	PRIMARY KEY(`uuid`)
);
CREATE TABLE IF NOT EXISTS `inodes` (
	`uuid`	TEXT NOT NULL,
	`type`	TEXT NOT NULL,
	`hash`	TEXT NOT NULL,
	`original_path`	TEXT NOT NULL,
	`target_path`	TEXT NOT NULL,
	`size`	INTEGER NOT NULL,
	`user`	TEXT NOT NULL,
	`group`	TEXT NOT NULL,
	`mode`	TEXT NOT NULL,
	`mod_time`	INTEGER NOT NULL,
	PRIMARY KEY(`uuid`)
);
CREATE TABLE IF NOT EXISTS `blobs` (
	`hash`	TEXT NOT NULL,
	`volume_uuid`	TEXT NOT NULL,
	PRIMARY KEY(`hash`)
);
CREATE INDEX IF NOT EXISTS `idx_inodes_user` ON `inodes` (
	`user`	ASC
);
CREATE INDEX IF NOT EXISTS `idx_inodes_type` ON `inodes` (
	`type`	ASC
);
CREATE INDEX IF NOT EXISTS `idx_inodes_target_path` ON `inodes` (
	`target_path`	ASC
);
CREATE INDEX IF NOT EXISTS `idx_inodes_size` ON `inodes` (
	`size`	ASC
);
CREATE INDEX IF NOT EXISTS `idx_inodes_original_path` ON `inodes` (
	`original_path`	ASC
);
CREATE INDEX IF NOT EXISTS `idx_inodes_mod_time` ON `inodes` (
	`mod_time`	ASC
);
CREATE INDEX IF NOT EXISTS `idx_inodes_hash` ON `inodes` (
	`hash`	ASC
);
CREATE INDEX IF NOT EXISTS `idx_inodes_group` ON `inodes` (
	`group`	ASC
);
COMMIT;
