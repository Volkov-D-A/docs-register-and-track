import dayjs from 'dayjs';

export const buildCitizenAppealEditFormValues = (record: any) => ({
    registrationNumber: record.registrationNumber,
    registrationDate: record.registrationDate ? dayjs(record.registrationDate) : null,
    appealDate: record.appealDate ? dayjs(record.appealDate) : null,
    applicantFullName: record.applicantFullName,
    registrationAddress: record.registrationAddress,
    appealType: record.appealType,
    applicantCategory: record.applicantCategory,
    appealPagesCount: record.appealPagesCount,
    attachmentPagesCount: record.attachmentPagesCount,
    hasEnvelope: !!record.hasEnvelope,
    receivedFromPos: !!record.receivedFromPos,
    content: record.content,
    correspondents: (record.correspondents || []).map((item: any) => ({
        registrationNumber: item.registrationNumber,
        registrationDate: item.registrationDate ? dayjs(item.registrationDate) : null,
        correspondentName: item.correspondentName,
    })),
    resolutions: (record.resolutions?.length ? record.resolutions : [{}]).map((item: any) => ({
        resolution: item.resolution || '',
        resolutionAuthor: item.resolutionAuthor || '',
        resolutionExecutors: item.resolutionExecutors ? item.resolutionExecutors.split('; ').filter((s: string) => s) : [],
    })),
});
