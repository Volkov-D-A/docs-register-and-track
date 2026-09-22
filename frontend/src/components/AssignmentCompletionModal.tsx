import React, { useEffect, useState } from 'react';
import { Modal, Input, Button, Typography, App } from 'antd';
import { UploadOutlined } from '@ant-design/icons';
import { formatAppError } from '../utils/appError';
import type { dto } from '../../wailsjs/go/models';
import { AttachmentUploadSummary } from './AttachmentUploadSummary';
import { emitAssignmentsChanged } from '../events/assignmentEvents';

const { TextArea } = Input;
const { Text } = Typography;

interface AssignmentCompletionModalProps {
    open: boolean;
    assignmentId: string;
    documentId: string;
    initialReport?: string;
    onCancel: () => void;
    onSuccess: () => void;
}

const AssignmentCompletionModal: React.FC<AssignmentCompletionModalProps> = ({
    open,
    assignmentId,
    documentId,
    initialReport = '',
    onCancel,
    onSuccess,
}) => {
    const { message } = App.useApp();
    const [reportText, setReportText] = useState('');
    const [submitting, setSubmitting] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [uploadResult, setUploadResult] = useState<dto.AttachmentUploadResult | null>(null);
    const [attachmentsEnabled, setAttachmentsEnabled] = useState(true);

    useEffect(() => {
        if (!open) {
            setReportText('');
            setAttachmentsEnabled(true);
            return;
        }

        setUploadResult(null);
        setReportText(initialReport);

        let isMounted = true;
        const loadSetting = async () => {
            try {
                const { IsAssignmentCompletionAttachmentsEnabled } = await import('../../wailsjs/go/services/SettingsService');
                const enabled = await IsAssignmentCompletionAttachmentsEnabled();
                if (isMounted) {
                    setAttachmentsEnabled(enabled);
                }
            } catch {
                if (isMounted) {
                    setAttachmentsEnabled(true);
                }
            }
        };

        loadSetting();
        return () => {
            isMounted = false;
        };
    }, [open, initialReport, assignmentId]);

    const handleSubmit = async () => {
        if (submitting) {
            return;
        }
        if (!reportText.trim()) {
            message.error('Введите отчет об исполнении');
            return;
        }

        setSubmitting(true);
        try {
            const { UpdateStatus } = await import('../../wailsjs/go/services/AssignmentService');

            await UpdateStatus(assignmentId, 'completed', reportText.trim());
            message.success('Поручение исполнено');
            emitAssignmentsChanged({ documentId });
            onSuccess();
        } catch (err: unknown) {
            message.error(formatAppError(err));
        } finally {
            setSubmitting(false);
        }
    };

    const addAttachments = async () => {
        if (uploading) return;
        setUploading(true);
        try {
            const { UploadForAssignment } = await import('../../wailsjs/go/services/AttachmentService');
            const uploaded = await UploadForAssignment(assignmentId);
            if (uploaded.items.length > 0) {
                setUploadResult(uploaded);
                emitAssignmentsChanged({ documentId });
            }
        } catch (err: unknown) {
            message.error(formatAppError(err));
            emitAssignmentsChanged({ documentId });
        } finally {
            setUploading(false);
        }
    };

    return (
        <Modal
            title="Отчет об исполнении"
            open={open}
            onCancel={onCancel}
            onOk={handleSubmit}
            okText="Отметить исполненным"
            confirmLoading={submitting}
            destroyOnHidden
        >
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                <TextArea
                    rows={4}
                    value={reportText}
                    onChange={e => setReportText(e.target.value)}
                    placeholder="Введите результат выполнения поручения..."
                />

                {attachmentsEnabled && (
                    <div>
                        <Button icon={<UploadOutlined />} loading={uploading} onClick={() => void addAttachments()}>Добавить файлы</Button>
                        <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
                            Файлы прикрепляются сразу после выбора.
                        </Text>
                        {uploadResult && <AttachmentUploadSummary result={uploadResult} />}
                    </div>
                )}
            </div>
        </Modal>
    );
};

export default AssignmentCompletionModal;
