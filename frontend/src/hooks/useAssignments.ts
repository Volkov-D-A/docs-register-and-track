import { useCallback, useEffect, useState } from 'react';
import { App } from 'antd';
import { useDocumentKindAccess } from './useDocumentKindAccess';
import { formatAppError } from '../utils/appError';

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
        if (!documentId || !canManageAssignments) {
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
    }, [canManageAssignments, documentId]);

    useEffect(() => {
        void load();
    }, [load]);

    const deleteAssignment = useCallback(async (id: string) => {
        try {
            const { Delete } = await import('../../wailsjs/go/services/AssignmentService');
            await Delete(id);
            message.success('Поручение удалено');
            await load();
        } catch (error: unknown) {
            message.error(formatAppError(error));
        }
    }, [load, message]);

    const updateStatus = useCallback(async (id: string, status: string, report = '') => {
        try {
            const { UpdateStatus } = await import('../../wailsjs/go/services/AssignmentService');
            await UpdateStatus(id, status, report);
            message.success('Статус поручения обновлён');
            await load();
        } catch (error: unknown) {
            message.error(formatAppError(error));
        }
    }, [load, message]);

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
