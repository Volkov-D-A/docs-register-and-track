package mocks

import "github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"

var (
	_ ports.AttachmentStore     = (*AttachmentStore)(nil)
	_ ports.JournalStore        = (*JournalStore)(nil)
	_ ports.DashboardStore      = (*DashboardStore)(nil)
	_ ports.OutgoingDocStore    = (*OutgoingDocStore)(nil)
	_ ports.LinkStore           = (*LinkStore)(nil)
	_ ports.UserStore           = (*UserStore)(nil)
	_ ports.FileStorage         = (*FileStorage)(nil)
	_ ports.AssignmentStore     = (*AssignmentStore)(nil)
	_ ports.ReferenceStore      = (*ReferenceStore)(nil)
	_ ports.DepartmentStore     = (*DepartmentStore)(nil)
	_ ports.IncomingDocStore    = (*IncomingDocStore)(nil)
	_ ports.NomenclatureStore   = (*NomenclatureStore)(nil)
	_ ports.AcknowledgmentStore = (*AcknowledgmentStore)(nil)
	_ ports.SettingsStore       = (*SettingsStore)(nil)
)
