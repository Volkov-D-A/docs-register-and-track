package backup

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Refuse shared targets before taking any irreversible step. UUID object names
// include orphaned uploads created by Docflow before a failed database commit.
func (s *Service) requireDedicatedTarget(ctx context.Context, m Manifest) error {
	allowed := map[string]bool{}
	for _, name := range strings.Fields(`schema_migrations departments users user_system_permissions nomenclature document_permissions department_nomenclature organizations resolution_executors documents document_command_idempotency document_correspondent_registrations document_resolutions incoming_document_details outgoing_document_details citizen_appeal_details administrative_order_details administrative_order_acknowledgment_people document_links assignment_series assignment_series_co_executors assignments assignment_co_executors attachments storage_statistics storage_statistics_mutations system_settings acknowledgments acknowledgment_users document_journal admin_audit_log user_events user_substitutions event_outbox server_sessions backup_settings backup_jobs backup_audit`) {
		allowed[name] = true
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT n.nspname,COALESCE(owner.relname,c.relname) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace LEFT JOIN pg_depend d ON c.relkind='S' AND d.classid='pg_class'::regclass AND d.objid=c.oid AND d.refclassid='pg_class'::regclass AND d.refobjsubid>0 AND d.deptype IN ('a','i') LEFT JOIN pg_class owner ON owner.oid=d.refobjid AND owner.relnamespace=c.relnamespace WHERE n.nspname NOT LIKE 'pg_%' AND n.nspname<>'information_schema' AND c.relkind IN ('r','p','v','m','S','f')`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var schema, name string
		if err = rows.Scan(&schema, &name); err != nil {
			rows.Close()
			return err
		}
		if schema != "public" || !allowed[name] {
			rows.Close()
			return fmt.Errorf("восстановление поддерживает только выделенную БД Docflow")
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var foreign bool
	if err = s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='public' AND p.proname<>'ensure_active_administrator_exists') OR EXISTS(SELECT 1 FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname='public' AND t.typtype IN ('e','d'))`).Scan(&foreign); err != nil {
		return err
	}
	if foreign {
		return fmt.Errorf("целевая БД содержит посторонние функции или типы")
	}
	referenced := map[string]bool{}
	rows, err = s.DB.QueryContext(ctx, "SELECT storage_path FROM attachments")
	if err != nil {
		return err
	}
	for rows.Next() {
		var key string
		if err = rows.Scan(&key); err != nil {
			rows.Close()
			return err
		}
		referenced[key] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, object := range m.Objects {
		if referenced[object.Key] {
			continue
		}
		if len(object.Key) >= 36 {
			id, err := uuid.Parse(object.Key[:36])
			if err == nil && id.String() == object.Key[:36] && (len(object.Key) == 36 || object.Key[36] == '.') && !strings.ContainsAny(object.Key, "/\\") {
				continue
			}
		}
		return fmt.Errorf("bucket содержит объекты вне данных Docflow; общее хранилище не поддерживается")
	}
	return nil
}
