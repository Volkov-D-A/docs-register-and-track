import {
    DOCUMENT_KIND_INCOMING_LETTER,
    DOCUMENT_KIND_OUTGOING_LETTER,
} from '../constants/documentKinds';

export type DocumentViewTabKey =
    | 'info'
    | 'assignments'
    | 'files'
    | 'links'
    | 'acknowledgments'
    | 'journal';

export type DocumentViewConfig = {
    tabs: DocumentViewTabKey[];
    createRelatedLabel: string;
};

const defaultViewConfig: DocumentViewConfig = {
    tabs: ['info', 'assignments', 'files', 'links', 'acknowledgments', 'journal'],
    createRelatedLabel: 'Создать связанный документ',
};

const documentViewConfigs: Record<string, DocumentViewConfig> = {
    [DOCUMENT_KIND_INCOMING_LETTER]: defaultViewConfig,
    [DOCUMENT_KIND_OUTGOING_LETTER]: defaultViewConfig,
};

export const getDocumentViewConfig = (kindCode: string): DocumentViewConfig => (
    documentViewConfigs[kindCode] || defaultViewConfig
);
