ALTER TABLE attachments ADD COLUMN content BYTEA;

ALTER TABLE attachments DROP COLUMN filepath;