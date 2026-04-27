import dayjs from 'dayjs';

export const buildIncomingLetterEditFormValues = (record: any) => ({
    documentTypeId: record.documentTypeId,
    correspondents: (record.correspondents?.length ? record.correspondents : [{}]).map((item: any) => ({
        registrationNumber: item.registrationNumber,
        registrationDate: item.registrationDate ? dayjs(item.registrationDate) : null,
        correspondentName: item.correspondentName,
    })),
    content: record.content,
    pagesCount: record.pagesCount,
    senderSignatory: record.senderSignatory,
    resolution: record.resolution || '',
    resolutionAuthor: record.resolutionAuthor || '',
    resolutionExecutors: record.resolutionExecutors ? record.resolutionExecutors.split('; ').filter((s: string) => s) : [],
});
