import React, { useEffect, useState } from 'react';
import { Modal, Input, Upload, Button, Typography, App } from 'antd';
import { UploadOutlined } from '@ant-design/icons';
import type { UploadFile, UploadProps } from 'antd';

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

const fileToBase64 = (file: File): Promise<string> => new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = () => reject(new Error('Ошибка чтения файла'));
});

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
    const [fileList, setFileList] = useState<UploadFile[]>([]);
    const [submitting, setSubmitting] = useState(false);

    useEffect(() => {
        if (open) {
            setReportText(initialReport);
            setFileList([]);
            return;
        }

        setReportText('');
        setFileList([]);
    }, [open, initialReport]);

    const handleSubmit = async () => {
        if (!reportText.trim()) {
            message.error('Введите отчет об исполнении');
            return;
        }

        setSubmitting(true);
        try {
            const { Upload: UploadAttachment } = await import('../../wailsjs/go/services/AttachmentService');
            const { UpdateStatus } = await import('../../wailsjs/go/services/AssignmentService');

            for (const file of fileList) {
                const originalFile = file.originFileObj;
                if (!originalFile) {
                    continue;
                }

                const base64Content = await fileToBase64(originalFile as File);
                await UploadAttachment(documentId, originalFile.name, base64Content);
            }

            await UpdateStatus(assignmentId, 'completed', reportText.trim());
            message.success('Поручение исполнено');
            onSuccess();
        } catch (err: any) {
            message.error(err?.message || String(err));
        } finally {
            setSubmitting(false);
        }
    };

    const uploadProps: UploadProps = {
        multiple: true,
        beforeUpload: () => false,
        fileList,
        onChange: ({ fileList: nextFileList }) => setFileList(nextFileList),
        onRemove: (file) => {
            setFileList(current => current.filter(item => item.uid !== file.uid));
            return true;
        },
    };

    return (
        <Modal
            title="Отчет об исполнении"
            open={open}
            onCancel={onCancel}
            onOk={handleSubmit}
            okText="Исполнено"
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

                <div>
                    <Upload {...uploadProps}>
                        <Button icon={<UploadOutlined />}>Добавить файлы</Button>
                    </Upload>
                    <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
                        Файлы будут прикреплены к документу при завершении поручения.
                    </Text>
                </div>
            </div>
        </Modal>
    );
};

export default AssignmentCompletionModal;
