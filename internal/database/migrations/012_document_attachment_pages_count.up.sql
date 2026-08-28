-- Общие счетчики для всех видов: pages_count хранит листы основного
-- документа, attachment_pages_count — листы приложения.
ALTER TABLE documents
    ADD COLUMN attachment_pages_count INT NOT NULL DEFAULT 0
        CHECK (attachment_pages_count >= 0);

ALTER TABLE citizen_appeal_details
    DROP COLUMN appeal_pages_count,
    DROP COLUMN attachment_pages_count;
