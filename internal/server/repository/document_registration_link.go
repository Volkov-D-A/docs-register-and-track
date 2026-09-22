package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/effects"
)

func createRegistrationLinkTx(tx *sql.Tx, outbox *OutboxRepository, documentID, actorID uuid.UUID, link *models.DocumentRegistrationLink) error {
	if link == nil {
		return nil
	}
	if outbox == nil {
		return ErrOutboxNotConfigured
	}
	sourceID, targetID := link.DocumentID, documentID
	if link.LinkType == "order_amends" || link.LinkType == "order_cancels" {
		sourceID, targetID = documentID, link.DocumentID
	}
	id := uuid.New()
	var createdAt time.Time
	if err := tx.QueryRow(`INSERT INTO document_links (id, source_document_id, target_document_id, link_type, created_by)
		VALUES ($1, $2, $3, $4, $5) RETURNING created_at`, id, sourceID, targetID, link.LinkType, actorID).Scan(&createdAt); err != nil {
		return err
	}
	if link.LinkType == "order_cancels" {
		if err := cancelAdministrativeOrderByLink(context.Background(), tx, targetID, createdAt); err != nil {
			return err
		}
	}
	for _, linkedID := range []uuid.UUID{sourceID, targetID} {
		event, err := effects.NewJournalOutboxEvent("link:"+id.String()+":LINK_CREATE:"+linkedID.String(), models.CreateJournalEntryRequest{
			DocumentID: linkedID, UserID: actorID, Action: "LINK_CREATE", Details: "Создана связь с другим документом",
		})
		if err != nil {
			return err
		}
		if err := outbox.EnqueueTx(tx, event); err != nil {
			return err
		}
	}
	return nil
}
