# Исходный Wails API

Снимок этапа 1, 8 сентября 2026. Это перечень текущих экспортов, а не целевой контракт.
UI: реальные CallExpression, разрешённые TypeScript checker (без test/spec).
«—» означает отсутствие прямого вызова в production UI. Go: обращения к методам вне тестов, разрешённые go/types по типу получателя
(включая передачу метода как callback). Служебные методы подлежат удалению.

## AcknowledgmentService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `Create` | `frontend/src/components/AcknowledgmentModal.tsx:54` | — |
| `Delete` | `frontend/src/components/AcknowledgmentList.tsx:60` | — |
| `GetAllActive` | `frontend/src/pages/DashboardPage.tsx:60` | — |
| `GetCurrentUserPendingByDocument` | `frontend/src/components/DocumentAcknowledgmentWorkflowPanel.tsx:30` | — |
| `GetList` | `frontend/src/components/AcknowledgmentList.tsx:39` | — |
| `GetPendingForCurrentUser` | `frontend/src/pages/DashboardPage.tsx:61` | — |
| `MarkConfirmed` | `frontend/src/components/DocumentAcknowledgmentWorkflowPanel.tsx:46` | — |
| `MarkViewed` | — | — |
| `SetSubstitutionStore` (служебный) | — | `internal/server/management.go:206` |

## AdminAuditLogService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetAll` | `frontend/src/features/settings/AuditLogTab.tsx:40` | — |
| `LogAction` | — | — |

## AdministrativeOrderService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `MarkAcknowledged` | `frontend/src/components/DocumentViewModal.tsx:123` | — |

## AssignmentService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `CancelSeries` | `frontend/src/components/AssignmentSeriesModal.tsx:106` | — |
| `Create` | `frontend/src/components/AssignmentModal.tsx:99` | — |
| `CreateSeries` | `frontend/src/components/AssignmentModal.tsx:87` | — |
| `Delete` | `frontend/src/hooks/useAssignments.ts:61`, `frontend/src/pages/AssignmentsPage.tsx:141` | — |
| `GetByID` | — | — |
| `GetList` | `frontend/src/hooks/useAssignments.ts:31`, `frontend/src/pages/AssignmentsPage.tsx:83` | — |
| `GetSeries` | `frontend/src/components/AssignmentSeriesModal.tsx:50` | — |
| `GetSeriesHistory` | `frontend/src/components/AssignmentSeriesModal.tsx:50` | — |
| `SetSubstitutionStore` (служебный) | — | `internal/server/management.go:197` |
| `Update` | `frontend/src/components/AssignmentModal.tsx:74` | — |
| `UpdateSeries` | `frontend/src/components/AssignmentSeriesModal.tsx:79` | — |
| `UpdateStatus` | `frontend/src/components/AssignmentCompletionModal.tsx:75`, `frontend/src/hooks/useAssignments.ts:73` | — |

## AttachmentService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `BulkDeleteOlderThan` | `frontend/src/features/settings/StorageTab.tsx:35` | — |
| `Delete` | `frontend/src/hooks/useAttachments.tsx:124` | — |
| `DownloadToDisk` | `frontend/src/hooks/useAttachments.tsx:84` | — |
| `GetAssignmentFiles` | `frontend/src/components/AssignmentSeriesModal.tsx:122` | — |
| `GetList` | `frontend/src/hooks/useAttachments.tsx:35` | — |
| `OpenFile` | `frontend/src/hooks/useAttachments.tsx:97` | — |
| `OpenFolder` | `frontend/src/hooks/useAttachments.tsx:103` | — |
| `ReconcileStorage` | `frontend/src/features/settings/StorageTab.tsx:51` | — |
| `Upload` | `frontend/src/hooks/useAttachments.tsx:70` | — |
| `UploadForAssignment` | `frontend/src/components/AssignmentCompletionModal.tsx:89` | — |

## AuthService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `ChangePassword` | `frontend/src/store/useAuthStore.ts:182` | — |
| `ChangeRequiredPassword` | `frontend/src/store/useAuthStore.ts:200` | — |
| `GetCurrentAuditInfo` | — | `internal/services/user_substitution_service.go:119`, `internal/services/user_substitution_service.go:142` |
| `GetCurrentUser` | — | `internal/services/helpers.go:118`, `internal/services/helpers.go:135`, `internal/services/settings.go:146` |
| `GetCurrentUserID` | — | `internal/app/app.go:101` |
| `GetCurrentUserUUID` | — | `internal/services/auth_service.go:339`, `internal/services/auth_service.go:441` |
| `GetSessionState` | `frontend/src/hooks/useSessionEvents.ts:14`, `frontend/src/store/useAuthStore.ts:126`, `frontend/src/store/useAuthStore.ts:183`, `frontend/src/store/useAuthStore.ts:187` | `internal/services/auth_service.go:468`, `internal/services/auth_service.go:473` |
| `HasAnySystemPermission` | — | — |
| `HasSystemPermission` | — | `internal/services/document_kind_service.go:124` |
| `HasSystemPermissionFor` | — | `internal/services/auth_service.go:517`, `internal/services/auth_service.go:559`, `internal/services/auth_service.go:572`, `internal/services/auth_service.go:589` |
| `InitialSetup` | `frontend/src/pages/LoginPage.tsx:81` | — |
| `IsAuthenticated` | — | — |
| `Login` | `frontend/src/store/useAuthStore.ts:125` | — |
| `Logout` | `frontend/src/store/useAuthStore.ts:171` | — |
| `NeedsInitialSetup` | `frontend/src/pages/LoginPage.tsx:28` | — |
| `RequireAnySystemPermission` | — | — |
| `RequireAuthenticated` | — | `internal/services/reference_service.go:31`, `internal/services/user_substitution_service.go:101` |
| `RequireSystemPermission` | — | `internal/services/user_substitution_service.go:98` |
| `SetAccessStore` (служебный) | — | — |
| `SetOperationMetrics` (служебный) | — | `internal/app/app.go:98` |
| `SetServerAuth` (служебный) | — | `internal/app/app.go:123` |
| `SetSettingsStore` (служебный) | — | — |
| `UpdateProfile` | `frontend/src/store/useAuthStore.ts:219` | — |

## DashboardService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetActivity` | `frontend/src/pages/DashboardPage.tsx:53` | — |
| `SetOperationMetrics` (служебный) | — | `internal/server/management.go:239` |

## DepartmentService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `CreateDepartment` | `frontend/src/features/settings/DepartmentsTab.tsx:49` | — |
| `DeleteDepartment` | `frontend/src/features/settings/DepartmentsTab.tsx:70` | — |
| `GetAllDepartments` | `frontend/src/features/settings/DepartmentsTab.tsx:23`, `frontend/src/features/settings/UsersTab.tsx:130` | — |
| `SetServerClient` (служебный) | — | `internal/app/app.go:147` |
| `UpdateDepartment` | `frontend/src/features/settings/DepartmentsTab.tsx:46` | — |

## DocumentAccessAdminService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetUserAccessProfile` | `frontend/src/features/settings/UsersTab.tsx:104` | — |
| `SetServerClient` (служебный) | — | `internal/app/app.go:133` |
| `UpdateUserAccessProfile` | `frontend/src/features/settings/UsersTab.tsx:162`, `frontend/src/features/settings/UsersTab.tsx:179` | — |

## DocumentKindService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetCurrentAccessSummary` | `frontend/src/store/accessSummaryCache.ts:20` | — |
| `SetServerClient` (служебный) | — | `internal/app/app.go:135` |

## DocumentQueryService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetByID` | `frontend/src/hooks/useDocumentDetails.ts:24`, `frontend/src/pages/OrdersPage.tsx:212` | — |
| `GetList` | `frontend/src/components/DocumentLinks/LinksTab.tsx:109`, `frontend/src/hooks/useDocumentListPage.ts:73` | — |
| `SetOperationMetrics` (служебный) | — | `internal/app/app.go:139` |
| `SetServerClient` (служебный) | — | `internal/app/app.go:138` |

## DocumentRegistrationService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `CreateAdminDraft` | `frontend/src/features/settings/NomenclatureTab.tsx:104` | — |
| `Register` | `frontend/src/hooks/useDocumentRegistrationActions.ts:83` | — |
| `SetOperationLifecycle` (служебный) | — | `internal/app/app.go:141` |
| `SetOperationMetrics` (служебный) | — | `internal/app/app.go:142`, `internal/server/management.go:188` |
| `Update` | `frontend/src/hooks/useDocumentRegistrationActions.ts:109` | — |

## JournalService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetByDocumentID` | `frontend/src/components/JournalList.tsx:36` | — |
| `SetOperationLifecycle` (служебный) | — | — |

## LinkService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetDocumentFlow` | `frontend/src/components/DocumentLinks/LinkGraph.tsx:208` | — |
| `GetDocumentLinks` | `frontend/src/components/DocumentLinks/LinksTab.tsx:210` | — |
| `LinkDocuments` | `frontend/src/components/DocumentLinks/LinksTab.tsx:243`, `frontend/src/hooks/useDocumentRegistrationActions.ts:71`, `frontend/src/pages/OrdersPage.tsx:78` | — |
| `SetOperationLifecycle` (служебный) | — | — |
| `SetOperationMetrics` (служебный) | — | `internal/server/management.go:225` |
| `UnlinkDocument` | `frontend/src/components/DocumentLinks/LinksTab.tsx:256` | — |

## NomenclatureService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `Create` | `frontend/src/features/settings/NomenclatureTab.tsx:54` | — |
| `Delete` | `frontend/src/features/settings/NomenclatureTab.tsx:75` | — |
| `GetActiveForKind` | `frontend/src/hooks/useNomenclaturesForKind.ts:12` | — |
| `GetAll` | `frontend/src/features/settings/DepartmentsTab.tsx:24`, `frontend/src/features/settings/NomenclatureTab.tsx:31` | — |
| `SetServerClient` (служебный) | — | `internal/app/app.go:129` |
| `Update` | `frontend/src/features/settings/NomenclatureTab.tsx:50` | — |

## OutboxAdminService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetFailed` | `frontend/src/features/settings/OutboxTab.tsx:43` | — |
| `GetStats` | `frontend/src/features/settings/OutboxTab.tsx:43` | — |
| `Requeue` | `frontend/src/features/settings/OutboxTab.tsx:64` | — |

## ReferenceService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `CreateDocumentType` | — | — |
| `DeleteDocumentType` | — | — |
| `DeleteOrganization` | `frontend/src/features/settings/ReferenceDirectoriesTab.tsx:50` | — |
| `DeleteResolutionExecutor` | `frontend/src/features/settings/ReferenceDirectoriesTab.tsx:211` | — |
| `FindOrCreateOrganization` | `frontend/src/features/settings/SystemSettingsTab.tsx:50`, `frontend/src/hooks/useOrganizationSetup.ts:86` | — |
| `FindOrCreateResolutionExecutor` | — | — |
| `GetDocumentTypes` | — | — |
| `GetOrganizations` | `frontend/src/features/settings/ReferenceDirectoriesTab.tsx:21` | — |
| `GetResolutionExecutors` | `frontend/src/features/settings/ReferenceDirectoriesTab.tsx:182` | — |
| `MergeOrganizations` | `frontend/src/features/settings/ReferenceDirectoriesTab.tsx:63` | — |
| `SearchOrganizations` | `frontend/src/hooks/useReferenceSearch.ts:31` | — |
| `SearchResolutionExecutors` | `frontend/src/hooks/useReferenceSearch.ts:59` | — |
| `SetServerClient` (служебный) | — | `internal/app/app.go:131` |
| `UpdateDocumentType` | — | — |
| `UpdateOrganization` | `frontend/src/features/settings/ReferenceDirectoriesTab.tsx:33` | — |
| `UpdateResolutionExecutor` | `frontend/src/features/settings/ReferenceDirectoriesTab.tsx:194` | — |

## ReleaseNoteService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetCurrent` | `frontend/src/App.tsx:115` | — |
| `MarkCurrentViewed` | `frontend/src/App.tsx:150` | — |

## SettingsService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetAll` | `frontend/src/features/settings/SystemSettingsTab.tsx:15`, `frontend/src/hooks/useOrganizationSetup.ts:42` | — |
| `GetAllowedFileTypes` | — | — |
| `GetMaxFileSize` | — | — |
| `GetMigrationStatus` | `frontend/src/features/settings/MigrationsTab.tsx:23` | — |
| `GetOrganizationName` | — | — |
| `GetOrganizationShortName` | `frontend/src/hooks/useBrandName.ts:17` | — |
| `IsAssignmentCompletionAttachmentsEnabled` | `frontend/src/components/AssignmentCompletionModal.tsx:45` | — |
| `RollbackMigration` | `frontend/src/features/settings/MigrationsTab.tsx:78` | — |
| `RunMigrations` | `frontend/src/features/settings/MigrationsTab.tsx:54` | — |
| `SetMigrationClient` (служебный) | — | `internal/app/app.go:121` |
| `SetServerClient` (служебный) | — | `internal/app/app.go:122` |
| `Update` | `frontend/src/features/settings/SystemSettingsTab.tsx:45`, `frontend/src/hooks/useOrganizationSetup.ts:84`, `frontend/src/hooks/useOrganizationSetup.ts:85` | — |

## StatisticsService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetAssignmentFilterOptions` | `frontend/src/features/statistics/AssignmentStatisticsTab.tsx:42` | — |
| `GetAssignmentReport` | `frontend/src/features/statistics/AssignmentStatisticsTab.tsx:54` | — |
| `GetAssignmentStatistics` | `frontend/src/features/statistics/AssignmentStatisticsTab.tsx:34` | — |
| `GetDocumentFilterOptions` | `frontend/src/features/statistics/DocumentStatisticsTab.tsx:45` | — |
| `GetDocumentReport` | `frontend/src/features/statistics/DocumentStatisticsTab.tsx:58` | — |
| `GetDocumentStatistics` | `frontend/src/features/statistics/DocumentStatisticsTab.tsx:37` | — |
| `GetStorageStatisticsStatus` | `frontend/src/features/statistics/SystemStatisticsTab.tsx:86`, `frontend/src/features/statistics/SystemStatisticsTab.tsx:138` | — |
| `GetSystemStatistics` | — | — |
| `RetryStorageStatisticsRefresh` | `frontend/src/features/statistics/SystemStatisticsTab.tsx:129` | — |
| `SetOperationLifecycle` (служебный) | — | — |
| `SetOperationMetrics` (служебный) | — | `internal/server/management.go:249` |

## SystemService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetBootstrapStatus` | `frontend/src/components/SystemBootstrapGate.tsx:16` | — |
| `Startup` (служебный) | — | `internal/app/app.go:210` |

## ThemeService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetTheme` | `frontend/src/theme/AppThemeProvider.tsx:88` | — |
| `SetTheme` | `frontend/src/theme/AppThemeProvider.tsx:119` | — |

## UserEventService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetCurrentUserEvents` | `frontend/src/components/layout/UserEventsButton.tsx:67` | — |
| `GetUnreadCount` | `frontend/src/components/layout/UserEventsButton.tsx:67`, `frontend/src/components/layout/UserEventsButton.tsx:84` | — |
| `MarkAllRead` | `frontend/src/components/layout/UserEventsButton.tsx:145` | — |
| `MarkDocumentRead` | `frontend/src/components/DocumentViewModal.tsx:63` | — |
| `MarkRead` | `frontend/src/components/layout/UserEventsButton.tsx:127` | — |

## UserService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `CreateUser` | `frontend/src/features/settings/UsersTab.tsx:172` | — |
| `GetAllUsers` | `frontend/src/features/settings/UsersTab.tsx:130` | — |
| `GetExecutors` | `frontend/src/components/AcknowledgmentModal.tsx:38`, `frontend/src/components/AssignmentModal.tsx:41`, `frontend/src/components/AssignmentSeriesModal.tsx:51`, `frontend/src/pages/AssignmentsPage.tsx:67` | — |
| `GetSubstitutionCandidates` | `frontend/src/pages/ProfilePage.tsx:57` | — |
| `ResetPassword` | `frontend/src/features/settings/UsersTab.tsx:224` | — |
| `SetServerClient` (служебный) | — | `internal/app/app.go:125` |
| `UpdateUser` | `frontend/src/features/settings/UsersTab.tsx:154` | — |

## UserSubstitutionService

| Метод | UI | Внутренние Go-обращения |
| --- | --- | --- |
| `GetMySubstitution` | `frontend/src/pages/ProfilePage.tsx:57` | — |
| `GetUserSubstitution` | `frontend/src/features/settings/UsersTab.tsx:105` | — |
| `SetServerClient` (служебный) | — | `internal/app/app.go:127` |
| `UpdateMySubstitution` | `frontend/src/pages/ProfilePage.tsx:118` | — |
| `UpdateUserSubstitution` | `frontend/src/features/settings/UsersTab.tsx:163`, `frontend/src/features/settings/UsersTab.tsx:180` | — |
