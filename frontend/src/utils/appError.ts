export interface AppErrorView {
    code: string;
    message: string;
    status?: number;
}

const DEFAULT_ERROR_MESSAGE = 'Произошла ошибка';

const CODE_MESSAGES: Record<string, string> = {
    UNAUTHORIZED: 'Требуется вход в систему',
    INVALID_CREDENTIALS: 'Неверный логин или пароль',
    USER_INACTIVE: 'Пользователь деактивирован',
    USER_LOCKED: 'Учетная запись заблокирована после 5 неверных попыток входа. Обратитесь к администратору для повторной активации.',
    FORBIDDEN: 'Недостаточно прав для выполнения действия',
    NOT_FOUND: 'Запрошенные данные не найдены',
    VALIDATION_ERROR: 'Проверьте заполнение формы',
    CONFLICT: 'Действие конфликтует с текущим состоянием данных',
    IDEMPOTENCY_CONFLICT: 'Повторный запрос отличается от исходного. Обновите форму и попробуйте снова',
    INTERNAL_ERROR: 'Произошла внутренняя ошибка',
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

        if (code || message || status) {
            return {
                code: code || (status === 404 ? 'NOT_FOUND' : 'UNKNOWN_ERROR'),
                message: message || (code ? CODE_MESSAGES[code] : undefined) || fallbackMessage || DEFAULT_ERROR_MESSAGE,
                status,
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
        message: raw || fallbackMessage || DEFAULT_ERROR_MESSAGE,
    };
};

export const formatAppError = (error: unknown, fallbackMessage = DEFAULT_ERROR_MESSAGE): string => {
    const appError = normalizeAppError(error, fallbackMessage);
    return appError.message || CODE_MESSAGES[appError.code] || fallbackMessage || DEFAULT_ERROR_MESSAGE;
};

export const getAppErrorCode = (error: unknown): string => normalizeAppError(error).code;
