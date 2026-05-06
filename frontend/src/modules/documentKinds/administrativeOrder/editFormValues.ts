import dayjs from 'dayjs';

export const buildAdministrativeOrderEditFormValues = (record: any) => ({
    orderDate: record.orderDate ? dayjs(record.orderDate) : undefined,
    title: record.title || record.content || '',
    executionController: record.executionController || '',
    executionDeadline: record.executionDeadline ? dayjs(record.executionDeadline) : undefined,
    isActive: record.isActive !== false,
    cancelledAt: record.cancelledAt ? dayjs(record.cancelledAt) : undefined,
    acknowledgmentFullNames: (record.acknowledgmentPeople || []).map((item: any) => item.fullName),
});
