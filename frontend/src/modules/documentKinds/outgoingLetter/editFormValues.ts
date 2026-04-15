import dayjs from 'dayjs';

export const buildOutgoingLetterEditFormValues = (record: any) => ({
    ...record,
    outgoingDate: dayjs(record.outgoingDate),
});
