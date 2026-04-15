import dayjs from 'dayjs';

export const buildIncomingLetterEditFormValues = (record: any) => ({
    documentTypeId: record.documentTypeId,
    senderOrgName: record.senderOrgName,
    outgoingNumberSender: record.outgoingNumberSender,
    outgoingDateSender: record.outgoingDateSender ? dayjs(record.outgoingDateSender) : null,
    content: record.content,
    pagesCount: record.pagesCount,
    senderSignatory: record.senderSignatory,
    intermediateNumber: record.intermediateNumber || '',
    intermediateDate: record.intermediateDate ? dayjs(record.intermediateDate) : null,
    resolution: record.resolution || '',
    resolutionAuthor: record.resolutionAuthor || '',
    resolutionExecutors: record.resolutionExecutors ? record.resolutionExecutors.split('; ').filter((s: string) => s) : [],
});
