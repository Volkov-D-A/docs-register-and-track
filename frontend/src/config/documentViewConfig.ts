import {
    DOCUMENT_KIND_CITIZEN_APPEAL,
    DOCUMENT_KIND_ADMINISTRATIVE_ORDER,
    DOCUMENT_KIND_INCOMING_LETTER,
    DOCUMENT_KIND_OUTGOING_LETTER,
} from '../constants/documentKinds';

export type DocumentViewTabKey =
    | 'info'
    | 'controls'
    | 'links'
    | 'journal';

export type DocumentViewConfig = {
    tabs: DocumentViewTabKey[];
    createRelatedLabel: string;
};

const defaultViewConfig: DocumentViewConfig = {
    tabs: ['info', 'controls', 'links', 'journal'],
    createRelatedLabel: 'Создать связанный документ',
};

const documentViewConfigs: Record<string, DocumentViewConfig> = {
    [DOCUMENT_KIND_INCOMING_LETTER]: defaultViewConfig,
    [DOCUMENT_KIND_OUTGOING_LETTER]: defaultViewConfig,
    [DOCUMENT_KIND_CITIZEN_APPEAL]: defaultViewConfig,
    [DOCUMENT_KIND_ADMINISTRATIVE_ORDER]: defaultViewConfig,
};

export const getDocumentViewConfig = (kindCode: string): DocumentViewConfig => (
    documentViewConfigs[kindCode] || defaultViewConfig
);
