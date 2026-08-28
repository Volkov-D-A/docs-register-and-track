-- Перед удалением общего счетчика возвращаем прежнюю структуру обращений.
ALTER TABLE citizen_appeal_details
    ADD COLUMN appeal_pages_count INT NOT NULL DEFAULT 1,
    ADD COLUMN attachment_pages_count INT NOT NULL DEFAULT 0;

UPDATE citizen_appeal_details ca
SET appeal_pages_count = d.pages_count,
    attachment_pages_count = d.attachment_pages_count
FROM documents d
WHERE d.id = ca.document_id
  AND d.kind = 'citizen_appeal';

ALTER TABLE documents
    DROP COLUMN attachment_pages_count;
