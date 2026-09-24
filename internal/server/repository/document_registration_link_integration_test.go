package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func createLinkedRegistration(t *testing.T, db *database.DB, kind models.DocumentKind, userID, nomID, orgID, key uuid.UUID, hash string, link *models.DocumentRegistrationLink) (uuid.UUID, error) {
	t.Helper()
	date := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	outbox := NewOutboxRepository(db)
	switch kind {
	case models.DocumentKindIncomingLetter:
		repo := NewIncomingDocumentRepository(db)
		repo.SetOutbox(outbox)
		doc, err := repo.CreateWithJournal(models.CreateIncomingDocRequest{CreatedBy: userID, NomenclatureID: nomID, IdempotencyKey: key, CommandHash: hash, Link: link, IncomingDate: date, DocumentTypeID: models.DocumentTypeLetter, Content: "linked", PagesCount: 1}, "CREATE", "Created %s")
		if err != nil {
			return uuid.Nil, err
		}
		return doc.ID, nil
	case models.DocumentKindOutgoingLetter:
		repo := NewOutgoingDocumentRepository(db)
		repo.SetOutbox(outbox)
		doc, err := repo.CreateWithJournal(models.CreateOutgoingDocRequest{CreatedBy: userID, NomenclatureID: nomID, IdempotencyKey: key, CommandHash: hash, Link: link, OutgoingDate: date, RecipientOrgID: orgID, DocumentTypeID: models.DocumentTypeLetter, Content: "linked", PagesCount: 1}, "CREATE", "Created %s")
		if err != nil {
			return uuid.Nil, err
		}
		return doc.ID, nil
	case models.DocumentKindCitizenAppeal:
		repo := NewCitizenAppealRepository(db)
		repo.SetOutbox(outbox)
		doc, err := repo.CreateWithJournal(models.CreateCitizenAppealDocRequest{CreatedBy: userID, NomenclatureID: nomID, IdempotencyKey: key, CommandHash: hash, Link: link, RegistrationDate: date, AppealDate: date, AppealType: "заявление", Content: "linked", PagesCount: 1}, "CREATE", "Created %s")
		if err != nil {
			return uuid.Nil, err
		}
		return doc.ID, nil
	default:
		repo := NewAdministrativeOrderRepository(db)
		repo.SetOutbox(outbox)
		doc, err := repo.CreateWithJournal(models.CreateAdministrativeOrderDocRequest{CreatedBy: userID, NomenclatureID: nomID, IdempotencyKey: key, CommandHash: hash, Link: link, OrderDate: date, Title: "linked", PagesCount: 1, IsActive: true}, "CREATE", "Created %s")
		if err != nil {
			return uuid.Nil, err
		}
		return doc.ID, nil
	}
}

func TestLinkedRegistrationAtomicityAndReplayIntegration(t *testing.T) {
	for _, kind := range []models.DocumentKind{models.DocumentKindIncomingLetter, models.DocumentKindOutgoingLetter, models.DocumentKindCitizenAppeal, models.DocumentKindAdministrativeOrder} {
		t.Run(string(kind), func(t *testing.T) {
			db := &database.DB{DB: integrationdb.Open(t)}
			userID, nomID, targetNomID, targetID, orgID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
			execSQL(t, db.DB, `INSERT INTO users(id,login,password_hash,full_name) VALUES($1,'linked-user','hash','Linked User')`, userID)
			execSQL(t, db.DB, `INSERT INTO organizations(id,name) VALUES($1,'Linked Org')`, orgID)
			execSQL(t, db.DB, `INSERT INTO nomenclature(id,name,index,year,kind_code,numbering_mode,next_number) VALUES($1,'New','LINK',2026,$2,'index_and_number',1)`, nomID, kind)
			execSQL(t, db.DB, `INSERT INTO nomenclature(id,name,index,year,kind_code) VALUES($1,'Existing','OLD',2026,'administrative_order')`, targetNomID)
			execSQL(t, db.DB, `INSERT INTO documents(id,kind,nomenclature_id,registration_number,registration_date,document_type,content,created_by) VALUES($1,'administrative_order',$2,'OLD-1','2026-09-01','Приказ','Existing',$3)`, targetID, targetNomID, userID)
			execSQL(t, db.DB, `INSERT INTO administrative_order_details(document_id,order_number,order_date,title) VALUES($1,'OLD-1','2026-09-01','Existing')`, targetID)
			key := uuid.New()
			link := &models.DocumentRegistrationLink{DocumentID: uuid.New(), LinkType: "related"}
			if kind == models.DocumentKindAdministrativeOrder {
				link.LinkType = "order_cancels"
			}
			assertRolledBack := func() {
				assertScalar(t, db.DB, `SELECT COUNT(*) FROM documents WHERE nomenclature_id=$1`, []any{nomID}, 0)
				assertScalar(t, db.DB, `SELECT next_number FROM nomenclature WHERE id=$1`, []any{nomID}, 1)
				assertScalar(t, db.DB, `SELECT COUNT(*) FROM document_command_idempotency WHERE idempotency_key=$1`, []any{key}, 0)
				assertScalar(t, db.DB, `SELECT COUNT(*) FROM document_links`, nil, 0)
				assertScalar(t, db.DB, `SELECT COUNT(*) FROM event_outbox`, nil, 0)
				var active bool
				require.NoError(t, db.QueryRow(`SELECT is_active FROM administrative_order_details WHERE document_id=$1`, targetID).Scan(&active))
				require.True(t, active, "failed registration must not cancel the old order")
			}
			// Foreign-key failure after document insertion must roll back the number too.
			_, err := createLinkedRegistration(t, db, kind, userID, nomID, orgID, key, "original", link)
			require.Error(t, err)
			assertRolledBack()
			link.DocumentID = targetID
			// Fail after the link (and order cancellation) to check the full transaction.
			execSQL(t, db.DB, `CREATE FUNCTION fail_link_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.payload->>'Action'='LINK_CREATE' THEN RAISE EXCEPTION 'test link audit failure'; END IF; RETURN NEW; END $$`)
			execSQL(t, db.DB, `CREATE TRIGGER fail_link_audit BEFORE INSERT ON event_outbox FOR EACH ROW EXECUTE FUNCTION fail_link_audit()`)
			_, err = createLinkedRegistration(t, db, kind, userID, nomID, orgID, key, "original", link)
			require.ErrorContains(t, err, "test link audit failure")
			assertRolledBack()
			execSQL(t, db.DB, `DROP TRIGGER fail_link_audit ON event_outbox`)
			id, err := createLinkedRegistration(t, db, kind, userID, nomID, orgID, key, "original", link)
			require.NoError(t, err)
			// Simulate a lost response: the whole command is retried with the same key.
			repeated, err := createLinkedRegistration(t, db, kind, userID, nomID, orgID, key, "original", link)
			require.NoError(t, err)
			require.Equal(t, id, repeated)
			assertScalar(t, db.DB, `SELECT COUNT(*) FROM documents WHERE nomenclature_id=$1`, []any{nomID}, 1)
			assertScalar(t, db.DB, `SELECT next_number FROM nomenclature WHERE id=$1`, []any{nomID}, 2)
			assertScalar(t, db.DB, `SELECT COUNT(*) FROM document_links`, nil, 1)
			assertScalar(t, db.DB, `SELECT COUNT(*) FROM event_outbox`, nil, 3)
			var source, target uuid.UUID
			require.NoError(t, db.QueryRow(`SELECT source_document_id,target_document_id FROM document_links`).Scan(&source, &target))
			if kind == models.DocumentKindAdministrativeOrder {
				require.Equal(t, id, source)
				require.Equal(t, targetID, target)
				var active bool
				require.NoError(t, db.QueryRow(`SELECT is_active FROM administrative_order_details WHERE document_id=$1`, targetID).Scan(&active))
				require.False(t, active)
			} else {
				require.Equal(t, targetID, source)
				require.Equal(t, id, target)
			}
			_, err = createLinkedRegistration(t, db, kind, userID, nomID, orgID, key, "changed-link-hash", &models.DocumentRegistrationLink{DocumentID: targetID, LinkType: "clarification"})
			require.Error(t, err)
			appErr, ok := models.AsAppError(err)
			require.True(t, ok)
			require.Equal(t, 409, appErr.StatusCode())
		})
	}
}
