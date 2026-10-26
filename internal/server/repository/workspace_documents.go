package repository

import (
	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// RecentDocuments applies each kind's read scope before taking the global top four.
func (r *WorkspaceRepository) RecentDocuments(scopes map[models.DocumentKind]models.DocumentAccessScope) ([]models.WorkspaceDocument, error) {
	items := []models.WorkspaceDocument{}
	args := []interface{}{}
	visibility, allowed := documentVisibilityScopes(scopes, &args)
	if allowed == "" {
		return items, nil
	}
	rows, err := r.db.Query(visibility+`, latest AS (
        SELECT d.* FROM documents d WHERE `+allowed+`
        ORDER BY d.created_at DESC, d.id DESC LIMIT 4
    )
    SELECT d.id, d.kind,
        d.registration_number,
        d.registration_date,
        d.created_at,
        COALESCE(NULLIF(CASE d.kind
            WHEN 'incoming_letter' THEN corr.names[1]
            WHEN 'outgoing_letter' THEN COALESCE(NULLIF(org.name, ''), NULLIF(out.addressee, ''))
            WHEN 'citizen_appeal' THEN appeal.applicant_full_name
            WHEN 'administrative_order' THEN ord.title
        END, ''), NULLIF(d.content, ''), '—'),
        corr.names
    FROM latest d
    LEFT JOIN outgoing_document_details out ON out.document_id = d.id
    LEFT JOIN organizations org ON org.id = out.recipient_org_id
    LEFT JOIN citizen_appeal_details appeal ON appeal.document_id = d.id
    LEFT JOIN administrative_order_details ord ON ord.document_id = d.id
    LEFT JOIN LATERAL (
        SELECT ARRAY(SELECT o.name
            FROM document_correspondent_registrations cr
            JOIN organizations o ON o.id = cr.correspondent_org_id
            WHERE cr.document_id = d.id AND d.kind = 'incoming_letter'
            ORDER BY cr.position, cr.created_at, cr.id) AS names
    ) corr ON true
    ORDER BY d.created_at DESC, d.id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item models.WorkspaceDocument
		if err := rows.Scan(&item.ID, &item.DocumentKind, &item.DocumentNumber, &item.DocumentDate,
			&item.RegisteredAt, &item.Description, pq.Array(&item.Correspondents)); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
