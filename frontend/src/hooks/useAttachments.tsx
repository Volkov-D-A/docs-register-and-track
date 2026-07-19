import React, { useCallback, useEffect, useRef, useState } from 'react';
import { App, Button, notification } from 'antd';
import { FileOutlined } from '@ant-design/icons';
import { useDocumentKindAccess } from './useDocumentKindAccess';
import { formatAppError } from '../utils/appError';
import { LatestRequest } from '../utils/latestRequest';

type UseAttachmentsOptions = {
    documentId: string;
    documentKind: string;
    readOnly?: boolean;
};

export const useAttachments = ({ documentId, documentKind, readOnly }: UseAttachmentsOptions) => {
    const { message } = App.useApp();
    const { hasAction, ready: accessReady } = useDocumentKindAccess();
    const canEdit = accessReady && !readOnly && hasAction(documentKind, 'upload');
    const [files, setFiles] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [deletingIds, setDeletingIds] = useState<Set<string>>(() => new Set());
    const [api, contextHolder] = notification.useNotification();
    const latestListRequestRef = useRef(new LatestRequest());
    const activeDocumentRef = useRef(documentId);
    activeDocumentRef.current = documentId;

    const loadFiles = useCallback(async () => {
        if (!documentId || activeDocumentRef.current !== documentId) {
            return;
        }
        setLoading(true);
        await latestListRequestRef.current.run(
            async () => {
                const { GetList } = await import('../../wailsjs/go/services/AttachmentService');
                return GetList(documentId);
            },
            {
                isRelevant: () => activeDocumentRef.current === documentId,
                onSuccess: (list) => setFiles(list || []),
                onError: (error) => {
                    console.error(error);
                    message.error(formatAppError(error, 'Не удалось загрузить файлы'));
                },
                onSettled: () => setLoading(false),
            },
        );
    }, [documentId, message]);

    useEffect(() => {
        const latestListRequest = latestListRequestRef.current;
        if (documentId) {
            setFiles([]);
            void loadFiles();
        } else {
            latestListRequest.invalidate();
            setFiles([]);
            setLoading(false);
        }

        return () => latestListRequest.invalidate();
    }, [documentId, loadFiles]);

    const uploadFile = useCallback(async () => {
        if (uploading) {
            return;
        }
        setUploading(true);
        try {
            const { Upload: uploadAttachment } = await import('../../wailsjs/go/services/AttachmentService');
            const uploaded = await uploadAttachment(documentId);
            if (uploaded.length > 0) { message.success('Файлы загружены'); await loadFiles(); }
        } catch (error: unknown) {
            message.error(formatAppError(error, 'Ошибка загрузки'));
        } finally {
            setUploading(false);
        }
    }, [documentId, loadFiles, message, uploading]);

    const downloadFile = useCallback(async (file: any) => {
        try {
            const { DownloadToDisk, OpenFile, OpenFolder } = await import('../../wailsjs/go/services/AttachmentService');

            message.loading({ content: 'Скачивание файла...', key: 'download' });
            const savedPath = await DownloadToDisk(file.id);
            message.success({ content: 'Файл сохранён', key: 'download' });

            const key = `open${Date.now()}`;
            api.open({
                title: 'Скачивание завершено',
                description: `Файл сохранён: ${savedPath}`,
                icon: <FileOutlined style={{ color: '#108ee9' }} />,
                key,
                duration: 0,
                actions: (
                    <div style={{ display: 'flex', gap: '8px' }}>
                        <Button type="primary" size="small" onClick={async () => {
                            await OpenFile(savedPath);
                            api.destroy(key);
                        }}>
                            Открыть
                        </Button>
                        <Button size="small" onClick={async () => {
                            await OpenFolder(savedPath);
                            api.destroy(key);
                        }}>
                            В папку
                        </Button>
                    </div>
                ),
            });
        } catch (error: unknown) {
            console.error(error);
            message.error({ content: formatAppError(error, 'Ошибка скачивания'), key: 'download' });
        }
    }, [api, message]);

    const deleteFile = useCallback(async (id: string) => {
        if (deletingIds.has(id)) {
            return;
        }
        setDeletingIds((current) => new Set(current).add(id));
        try {
            const { Delete } = await import('../../wailsjs/go/services/AttachmentService');
            await Delete(id);
            message.success('Файл удалён');
            await loadFiles();
        } catch (error: unknown) {
            message.error(formatAppError(error, 'Ошибка удаления'));
        } finally {
            setDeletingIds((current) => {
                const next = new Set(current);
                next.delete(id);
                return next;
            });
        }
    }, [deletingIds, loadFiles, message]);

    return {
        files,
        loading,
        uploading,
        deletingIds,
        accessReady,
        canEdit,
        contextHolder,
        uploadFile,
        downloadFile,
        deleteFile,
    };
};
