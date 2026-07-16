import React, { useEffect, useState } from 'react';
import { Modal, Input, Button, Typography, App } from 'antd';
import { UploadOutlined } from '@ant-design/icons';
import { formatAppError } from '../utils/appError';
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
    const [attachmentsEnabled, setAttachmentsEnabled] = useState(true);

    useEffect(() => {
        if (!open) {
            setReportText('');
            setAttachmentsEnabled(true);
            return;
        }

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
    }, [open, initialReport]);

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
        try {
            const { Upload: uploadAttachment } = await import('../../wailsjs/go/services/AttachmentService');
            const uploaded = await uploadAttachment(documentId);
            if (uploaded.length > 0) message.success('Файлы прикреплены');
        } catch (err: unknown) { message.error(formatAppError(err)); }
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
                        <Button icon={<UploadOutlined />} onClick={() => void addAttachments()}>Добавить файлы</Button>
                        <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
                            Файлы прикрепляются сразу после выбора.
                        </Text>
                    </div>
                )}
            </div>
        </Modal>
    );
};

export default AssignmentCompletionModal;
