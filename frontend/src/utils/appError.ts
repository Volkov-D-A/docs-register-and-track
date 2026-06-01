export interface AppErrorView {
    code: string;
    message: string;
    status?: number;
}

const DEFAULT_ERROR_MESSAGE = 'Не удалось выполнить действие';

const GENERIC_ACTION = 'Повторите попытку или обратитесь к администратору, если ошибка повторяется.';

const CODE_COPY: Record<string, { message: string; action: string; allowDetail?: boolean }> = {
    UNAUTHORIZED: {
        message: 'Требуется вход в систему',
        action: 'Войдите снова и повторите действие.',
    },
    INVALID_CREDENTIALS: {
        message: 'Неверный логин или пароль',
        action: 'Проверьте данные и попробуйте снова.',
    },
    USER_INACTIVE: {
        message: 'Пользователь деактивирован',
        action: 'Обратитесь к администратору для восстановления доступа.',
    },
    USER_LOCKED: {
        message: 'Учетная запись заблокирована после 5 неверных попыток входа',
        action: 'Обратитесь к администратору для повторной активации.',
    },
    FORBIDDEN: {
        message: 'Недостаточно прав для выполнения действия',
        action: 'Обратитесь к администратору, если доступ нужен для работы.',
    },
    NOT_FOUND: {
        message: 'Запрошенные данные не найдены',
        action: 'Обновите страницу или вернитесь к списку.',
        allowDetail: true,
    },
    VALIDATION_ERROR: {
        message: 'Проверьте заполнение формы',
        action: 'Исправьте данные и повторите действие.',
        allowDetail: true,
    },
    CONFLICT: {
        message: 'Данные изменились или действие конфликтует с текущим состоянием',
        action: 'Обновите данные и повторите действие.',
        allowDetail: true,
    },
    IDEMPOTENCY_CONFLICT: {
        message: 'Повторный запрос отличается от исходного',
        action: 'Обновите форму и попробуйте снова.',
    },
    INTERNAL_ERROR: {
        message: 'Не удалось выполнить действие из-за внутренней ошибки',
        action: GENERIC_ACTION,
    },
    UNKNOWN_ERROR: {
        message: DEFAULT_ERROR_MESSAGE,
        action: GENERIC_ACTION,
    },
};

const isRecord = (value: unknown): value is Record<string, unknown> => (
    typeof value === 'object' && value !== null
);

const readString = (value: unknown): string | undefined => (
    typeof value === 'string' && value.trim() ? value.trim() : undefined
);

const readStatus = (value: unknown): number | undefined => (
    typeof value === 'number' && Number.isFinite(value) ? value : undefined
);

const codeFromStatus = (status?: number): string => {
    if (status === 401) {
        return 'UNAUTHORIZED';
    }
    if (status === 403) {
        return 'FORBIDDEN';
    }
    if (status === 404) {
        return 'NOT_FOUND';
    }
    if (status === 409) {
        return 'CONFLICT';
    }
    if (status && status >= 500) {
        return 'INTERNAL_ERROR';
    }
    return 'UNKNOWN_ERROR';
};

const parseSerializedError = (value: string): AppErrorView | null => {
    try {
        const parsed = JSON.parse(value);
        return normalizeAppError(parsed, '');
    } catch {
        return null;
    }
};

export const normalizeAppError = (error: unknown, fallbackMessage = DEFAULT_ERROR_MESSAGE): AppErrorView => {
    if (isRecord(error)) {
        const code = readString(error.code);
        const message = readString(error.message);
        const status = readStatus(error.status);

        if (code || status) {
            return {
                code: code || codeFromStatus(status),
                message: message || fallbackMessage || DEFAULT_ERROR_MESSAGE,
                status,
            };
        }

        if (message) {
            return {
                code: 'UNKNOWN_ERROR',
                message: fallbackMessage || DEFAULT_ERROR_MESSAGE,
            };
        }
    }

    const raw = error instanceof Error
        ? error.message
        : readString(error) || '';
    const serialized = raw ? parseSerializedError(raw) : null;
    if (serialized) {
        return {
            ...serialized,
            message: serialized.message || fallbackMessage || DEFAULT_ERROR_MESSAGE,
        };
    }

    return {
        code: 'UNKNOWN_ERROR',
        message: fallbackMessage || DEFAULT_ERROR_MESSAGE,
    };
};

const ensureSentence = (text: string): string => {
    const trimmed = text.trim();
    if (!trimmed) {
        return '';
    }
    return /[.!?…]$/.test(trimmed) ? trimmed : `${trimmed}.`;
};

export const formatAppError = (error: unknown, fallbackMessage = DEFAULT_ERROR_MESSAGE): string => {
    const appError = normalizeAppError(error, fallbackMessage);
    const copy = CODE_COPY[appError.code] || CODE_COPY.UNKNOWN_ERROR;
    const detail = copy.allowDetail ? appError.message : '';
    const message = detail || copy.message || fallbackMessage || DEFAULT_ERROR_MESSAGE;

    return `${ensureSentence(message)} ${ensureSentence(copy.action)}`.trim();
};

export const getAppErrorCode = (error: unknown): string => normalizeAppError(error).code;
