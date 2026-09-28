package repository

import (
	"fmt"
	"strings"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// SearchDocuments reads current card fields, so edits and organization renames
// become searchable in the same transaction without a second index to synchronize.
func (r *DocumentRepository) SearchDocuments(request dto.DocumentSearchRequest, scopes map[models.DocumentKind]models.DocumentAccessScope) (*dto.DocumentSearchResult, error) {
	result := &dto.DocumentSearchResult{Items: []dto.DocumentSearchItem{}, Page: request.Page, PageSize: request.PageSize}
	args := []interface{}{request.Query}
	argIdx := 2
	allowed := []string{}
	for _, spec := range models.AllDocumentKindSpecs() {
		scope, ok := scopes[spec.Code]
		if !ok {
			continue
		}
		where := []string{fmt.Sprintf("d.kind = $%d", argIdx)}
		args = append(args, string(spec.Code))
		argIdx++
		applyDocumentListAccess(&where, &args, &argIdx, scope)
		allowed = append(allowed, "("+strings.Join(where, " AND ")+")")
	}
	if len(allowed) == 0 {
		return result, nil
	}
	limit, offset := argIdx, argIdx+1
	args = append(args, request.PageSize, (request.Page-1)*request.PageSize)
	// plainto_tsquery requires all meaningful words, regardless of which card
	// field contains them. Simple configuration also permits stop-word-only queries.
	query := `WITH query_config AS (
        SELECT CASE WHEN numnode(plainto_tsquery('russian', replace(lower($1), 'ё', 'е'))) = 0
            THEN 'simple'::regconfig ELSE 'russian'::regconfig END AS config
    ), search_query AS (
        SELECT config, plainto_tsquery(config, replace(lower($1), 'ё', 'е')) AS terms FROM query_config
    ), fields AS (
        SELECT d.id, d.kind, d.registration_number, d.registration_date, d.created_at,
            CASE WHEN d.kind = 'administrative_order' THEN coalesce(ord.title, '') ELSE d.content END AS content,
            coalesce(res.text, '') AS resolution,
            CASE WHEN d.kind = 'outgoing_letter' THEN coalesce(org.name, '') ELSE coalesce(corr.names, '') END AS correspondent,
            CASE d.kind
                WHEN 'incoming_letter' THEN coalesce(inc.sender_signatory, '')
                WHEN 'outgoing_letter' THEN concat_ws('; ', nullif(out.sender_signatory, ''), nullif(out.addressee, ''))
                ELSE '' END AS person
        FROM documents d
        LEFT JOIN incoming_document_details inc ON inc.document_id = d.id
        LEFT JOIN outgoing_document_details out ON out.document_id = d.id
        LEFT JOIN organizations org ON org.id = out.recipient_org_id
        LEFT JOIN administrative_order_details ord ON ord.document_id = d.id
        LEFT JOIN LATERAL (
            SELECT string_agg(dr.resolution, '; ' ORDER BY dr.position, dr.id) AS text
            FROM document_resolutions dr WHERE dr.document_id = d.id
        ) res ON true
        LEFT JOIN LATERAL (
            SELECT string_agg(o.name, '; ' ORDER BY cr.position, cr.id) AS names
            FROM document_correspondent_registrations cr
            JOIN organizations o ON o.id = cr.correspondent_org_id
            WHERE cr.document_id = d.id
        ) corr ON true
        WHERE ` + strings.Join(allowed, " OR ") + `
    ), vectors AS (
        SELECT f.*, q.terms,
            setweight(to_tsvector(q.config, replace(lower(content), 'ё', 'е')), 'A') ||
            setweight(to_tsvector(q.config, replace(lower(resolution), 'ё', 'е')), 'B') ||
            setweight(to_tsvector(q.config, replace(lower(correspondent), 'ё', 'е')), 'C') ||
            setweight(to_tsvector(q.config, replace(lower(person), 'ё', 'е')), 'C') AS vector
        FROM fields f CROSS JOIN search_query q
    ), matched AS MATERIALIZED (
        SELECT id, kind, registration_number, registration_date, created_at,
            content, resolution, correspondent, person,
            ts_rank('{0.1,0.3,0.6,1.0}'::real[], vector, terms, 32) AS relevance
        FROM vectors WHERE vector @@ terms
    ), page AS (
        SELECT * FROM matched ORDER BY relevance DESC, created_at DESC, id DESC
        LIMIT $` + fmt.Sprint(limit) + ` OFFSET $` + fmt.Sprint(offset) + `
    )
    SELECT totals.count, coalesce(p.id::text, ''), coalesce(p.kind, ''),
        coalesce(p.registration_number, ''), coalesce(p.registration_date, DATE '1970-01-01'),
        coalesce(p.content, ''), coalesce(p.resolution, ''), coalesce(p.correspondent, ''),
        coalesce(p.person, ''), coalesce(p.relevance, 0)
    FROM (SELECT count(*) FROM matched) totals LEFT JOIN page p ON true
    ORDER BY p.relevance DESC, p.created_at DESC, p.id DESC`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("search documents: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item dto.DocumentSearchItem
		if err := rows.Scan(&result.TotalCount, &item.ID, &item.KindCode, &item.RegistrationNumber,
			&item.RegistrationDate, &item.Content, &item.Resolution, &item.Correspondent, &item.Person, &item.Relevance); err != nil {
			return nil, err
		}
		if item.ID != "" {
			result.Items = append(result.Items, item)
		}
	}
	return result, rows.Err()
}
