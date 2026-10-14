package mocks

import "github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"

var (
	_ ports.JournalStore      = (*JournalStore)(nil)
	_ ports.OutgoingDocStore  = (*OutgoingDocStore)(nil)
	_ ports.LinkStore         = (*LinkStore)(nil)
	_ ports.UserStore         = (*UserStore)(nil)
	_ ports.ReferenceStore    = (*ReferenceStore)(nil)
	_ ports.IncomingDocStore  = (*IncomingDocStore)(nil)
	_ ports.NomenclatureStore = (*NomenclatureStore)(nil)
	_ ports.SettingsStore     = (*SettingsStore)(nil)
)
