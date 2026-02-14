ALTER TABLE attachments ADD COLUMN filepath TEXT NOT NULL DEFAULT '';

ALTER TABLE attachments DROP COLUMN content;