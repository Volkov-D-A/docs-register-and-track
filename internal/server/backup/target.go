package backup

import (
	"context"
	"fmt"
	"strings"
)

// Refuse shared database targets before taking any irreversible step.
func (s *Service) requireDedicatedTarget(ctx context.Context) error {
	allowed := map[string]bool{}
	for _, name := range strings.Fields(`schema_migrations departments users user_system_permissions nomenclature document_permissions department_nomenclature organizations resolution_executors documents document_command_idempotency document_correspondent_registrations document_resolutions incoming_document_details outgoing_document_details citizen_appeal_details administrative_order_details administrative_order_acknowledgment_people document_links deadline_executors document_deadline_controls document_deadline_series system_settings document_journal admin_audit_log user_events event_outbox server_sessions backup_settings`) {
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
	if err = s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='public' AND p.proname NOT IN ('ensure_active_administrator_exists','create_user_deadline_executor','deadline_executor_set_valid','verify_deadline_co_executors','restrict_deadline_co_executor_deletion')) OR EXISTS(SELECT 1 FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname='public' AND t.typtype IN ('e','d'))`).Scan(&foreign); err != nil {
		return err
	}
	if foreign {
		return fmt.Errorf("целевая БД содержит посторонние функции или типы")
	}
	return nil
}
