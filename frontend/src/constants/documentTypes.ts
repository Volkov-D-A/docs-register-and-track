export const DEFAULT_DOCUMENT_TYPE = 'Письмо';

export const DOCUMENT_TYPE_OPTIONS = [
    DEFAULT_DOCUMENT_TYPE,
    'Договор',
    'Акт',
    'Счёт',
    'Запрос',
    'Ответ',
    'Уведомление',
].map((name) => ({
    id: name,
    name,
    value: name,
    label: name,
}));
