import {
    DOCUMENT_KIND_INCOMING_LETTER,
    DOCUMENT_KIND_OUTGOING_LETTER,
    RegistrationKind,
    getDocumentKindShortLabel,
} from '../constants/documentKinds';

export type DocumentViewTabKey =
    | 'info'
    | 'assignments'
    | 'files'
    | 'links'
    | 'acknowledgments'
    | 'journal';

export type DocumentViewAction = {
    targetKind: RegistrationKind;
    label: string;
};

export type DocumentViewConfig = {
    tabs: DocumentViewTabKey[];
    restrictedTabs: DocumentViewTabKey[];
    footerActions: DocumentViewAction[];
};

const defaultViewConfig: DocumentViewConfig = {
    tabs: ['info', 'assignments', 'files', 'links', 'acknowledgments', 'journal'],
    restrictedTabs: ['info', 'files'],
    footerActions: [
        {
            targetKind: DOCUMENT_KIND_INCOMING_LETTER,
            label: `Создать связанный ${getDocumentKindShortLabel(DOCUMENT_KIND_INCOMING_LETTER).toLowerCase()}`,
        },
        {
            targetKind: DOCUMENT_KIND_OUTGOING_LETTER,
            label: `Создать связанный ${getDocumentKindShortLabel(DOCUMENT_KIND_OUTGOING_LETTER).toLowerCase()}`,
        },
    ],
};

const documentViewConfigs: Record<string, DocumentViewConfig> = {
    [DOCUMENT_KIND_INCOMING_LETTER]: defaultViewConfig,
    [DOCUMENT_KIND_OUTGOING_LETTER]: defaultViewConfig,
};

export const getDocumentViewConfig = (kindCode: string): DocumentViewConfig => (
    documentViewConfigs[kindCode] || defaultViewConfig
);
