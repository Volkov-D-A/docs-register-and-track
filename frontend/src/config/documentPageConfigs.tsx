import { DOCUMENT_KIND_ADMINISTRATIVE_ORDER, DOCUMENT_KIND_CITIZEN_APPEAL, DOCUMENT_KIND_INCOMING_LETTER, DOCUMENT_KIND_OUTGOING_LETTER } from '../constants/documentKinds';
import { incomingLetterPageConfig } from '../modules/documentKinds/incomingLetter';
import { outgoingLetterPageConfig } from '../modules/documentKinds/outgoingLetter';
import { citizenAppealPageConfig } from '../modules/documentKinds/citizenAppeal';
import { administrativeOrderPageConfig } from '../modules/documentKinds/administrativeOrder';

type DocumentPageConfig = {
    kindCode: string;
    title: string;
    tableClassName: string;
    registerModalTitle: string;
    getEditModalTitle: (record: any) => string;
    registerInitialValues: Record<string, unknown>;
    buildColumns: (params: {
        isExecutorOnly: boolean;
        openViewModal: (documentId: string) => void;
        onEdit: (record: any) => void;
    }) => any[];
};

export const documentPageConfigs: Record<string, DocumentPageConfig> = {
    [DOCUMENT_KIND_INCOMING_LETTER]: incomingLetterPageConfig,
    [DOCUMENT_KIND_OUTGOING_LETTER]: outgoingLetterPageConfig,
    [DOCUMENT_KIND_CITIZEN_APPEAL]: citizenAppealPageConfig,
    [DOCUMENT_KIND_ADMINISTRATIVE_ORDER]: administrativeOrderPageConfig,
};

export const getDocumentPageConfig = (kindCode: string): DocumentPageConfig => (
    documentPageConfigs[kindCode] || incomingLetterPageConfig
);
