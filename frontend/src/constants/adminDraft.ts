export const ADMIN_DRAFT_PLACEHOLDER = 'Черновик. Требуется заполнение.';

export const isAdminDraft = (record: any): boolean => (
    record?.content === ADMIN_DRAFT_PLACEHOLDER || record?.title === ADMIN_DRAFT_PLACEHOLDER
);
