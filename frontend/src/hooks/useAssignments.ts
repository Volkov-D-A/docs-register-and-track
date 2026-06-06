import { useCallback, useEffect, useState } from 'react';
import { App } from 'antd';
import { useDocumentKindAccess } from './useDocumentKindAccess';
import { formatAppError } from '../utils/appError';
import { emitAssignmentsChanged } from '../events/assignmentEvents';
import { isAssignmentUserEvent, onUserEventsReceived } from '../events/userEvents';

type UseAssignmentsOptions = {
    documentId: string;
    documentKind: string;
};

export const useAssignments = ({ documentId, documentKind }: UseAssignmentsOptions) => {
    const { message } = App.useApp();
    const { hasAction, ready: accessReady } = useDocumentKindAccess();
    const canManageAssignments = accessReady && hasAction(documentKind, 'assign');
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);

    const load = useCallback(async () => {
        if (!documentId || !accessReady) {
            setData([]);
            return;
        }
        setLoading(true);
        try {
            const { GetList } = await import('../../wailsjs/go/services/AssignmentService');
            const result = await GetList({ documentId, page: 1, pageSize: 100, showFinished: true, overdueOnly: false });
            setData(result?.items || []);
        } catch (error) {
            console.error(error);
        } finally {
            setLoading(false);
        }
    }, [accessReady, documentId]);

    useEffect(() => {
        void load();
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
        loading,
        accessReady,
        canManageAssignments,
        load,
        deleteAssignment,
        updateStatus,
    };
};
