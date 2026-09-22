import { useCallback, useEffect, useRef, useState } from 'react';
import { App } from 'antd';
import { useDocumentKindAccess } from './useDocumentKindAccess';
import { formatAppError } from '../utils/appError';
import { emitAssignmentsChanged } from '../events/assignmentEvents';
import { isAssignmentUserEvent, onUserEventsReceived } from '../events/userEvents';
import { dto, models } from '../../wailsjs/go/models';
import { CoalescedRequest } from '../utils/coalescedRequest';

// Bound automatic loading to 1,000 assignments per document.
const ASSIGNMENT_PAGE_SIZE = 100;
const MAX_ASSIGNMENT_PAGES = 10;
type AssignmentLoadResult = { items: dto.Assignment[]; warning: string | null };
const INCOMPLETE_ASSIGNMENTS = 'Список поручений неполный. Некоторые поручения и доступные действия могут не отображаться.';

type UseAssignmentsOptions = {
    documentId: string;
    documentKind: string;
};

export const useAssignments = ({ documentId, documentKind }: UseAssignmentsOptions) => {
    const { message } = App.useApp();
    const { hasAction, ready: accessReady } = useDocumentKindAccess();
    const canManageAssignments = accessReady && hasAction(documentKind, 'assign');
    const [data, setData] = useState<dto.Assignment[]>([]);
    const [loadWarning, setLoadWarning] = useState<string | null>(null);
    const activeDocumentRef = useRef(documentId);
    activeDocumentRef.current = documentId;
    const [loading, setLoading] = useState(false);
    const assignmentsRequestRef = useRef(new CoalescedRequest<AssignmentLoadResult>());

    const load = useCallback(async () => {
        if (!documentId || !accessReady) {
            assignmentsRequestRef.current.invalidate();
            setData([]);
            setLoadWarning(null);
            setLoading(false);
            return;
        }
        setLoading(true);
        return assignmentsRequestRef.current.refresh(async () => {
            const { GetList } = await import('../../wailsjs/go/services/AssignmentService');
            const items = new Map<string, dto.Assignment>();
            let initialTotal: number | undefined;
            for (let page = 1; page <= MAX_ASSIGNMENT_PAGES; page++) {
                if (activeDocumentRef.current !== documentId) break;
                try {
                    const result = await GetList(models.AssignmentFilter.createFrom({
                        documentId,
                        page,
                        pageSize: ASSIGNMENT_PAGE_SIZE,
                        showFinished: true,
                        overdueOnly: false,
                    }));
                    const previousSize = items.size;
                    for (const item of result.items || []) items.set(item.id, item);
                    initialTotal ??= result.totalCount;
                    if (result.totalCount !== initialTotal) {
                        return { items: [...items.values()], warning: `${INCOMPLETE_ASSIGNMENTS} Список изменился во время загрузки; обновите его.` };
                    }
                    if (items.size >= result.totalCount) {
                        return { items: [...items.values()], warning: null };
                    }
                    if (items.size === previousSize) {
                        return { items: [...items.values()], warning: `${INCOMPLETE_ASSIGNMENTS} Следующая страница не содержит новых записей; обновите список.` };
                    }
                } catch (error: unknown) {
                    return { items: [...items.values()], warning: `${INCOMPLETE_ASSIGNMENTS} ${formatAppError(error, 'Не удалось загрузить поручения')}` };
                }
            }
            return { items: [...items.values()], warning: `${INCOMPLETE_ASSIGNMENTS} Загружено не более ${MAX_ASSIGNMENT_PAGES * ASSIGNMENT_PAGE_SIZE} записей. Используйте раздел «Поручения» для поиска остальных.` };
        }, {
            onSuccess: (result) => { setData(result.items); setLoadWarning(result.warning); },
            onError: (error) => {
                setData([]);
                setLoadWarning(`${INCOMPLETE_ASSIGNMENTS} ${formatAppError(error, 'Не удалось загрузить поручения')}`);
            },
            onSettled: () => setLoading(false),
        });
    }, [accessReady, documentId]);

    useEffect(() => {
        const request = assignmentsRequestRef.current;
        setData([]);
        setLoadWarning(null);
        void load();
        return () => request.invalidate();
    }, [load]);

    useEffect(() => onUserEventsReceived((events) => {
        if (events.some((event) => isAssignmentUserEvent(event) && event.documentId === documentId)) {
            void load();
        }
    }), [documentId, load]);

    const deleteAssignment = useCallback(async (id: string) => {
        try {
            const { Delete } = await import('../../wailsjs/go/services/AssignmentService');
            await Delete(id);
            message.success('Поручение удалено');
            emitAssignmentsChanged({ documentId });
            await load();
        } catch (error: unknown) {
            message.error(formatAppError(error));
        }
    }, [documentId, load, message]);

    const updateStatus = useCallback(async (id: string, status: string, report = '') => {
        try {
            const { UpdateStatus } = await import('../../wailsjs/go/services/AssignmentService');
            await UpdateStatus(id, status, report);
            message.success('Статус поручения обновлён');
            emitAssignmentsChanged({ documentId });
            await load();
            return true;
        } catch (error: unknown) {
            message.error(formatAppError(error));
            return false;
        }
    }, [documentId, load, message]);

    return {
        data,
        loadWarning,
        loading,
        accessReady,
        canManageAssignments,
        load,
        deleteAssignment,
        updateStatus,
    };
};
