import React, { useState, useEffect } from 'react';
import { List, Button, Upload, message, Popconfirm, Avatar, Typography, Tooltip, notification } from 'antd';
import { UploadOutlined, DownloadOutlined, DeleteOutlined, FileOutlined, FilePdfOutlined, FileImageOutlined, FileWordOutlined } from '@ant-design/icons';
import type { UploadProps } from 'antd';
import { useAuthStore } from '../store/useAuthStore';

const { Text } = Typography;

interface FileListComponentProps {
    documentId: string;
    documentType: string;
    readOnly?: boolean;
}

const FileListComponent: React.FC<FileListComponentProps> = ({ documentId, documentType, readOnly }) => {
    const { currentRole } = useAuthStore();
    const canEdit = !readOnly && (currentRole === 'clerk' || currentRole === 'admin');

    const [files, setFiles] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [api, contextHolder] = notification.useNotification();

    const loadFiles = async () => {
        setLoading(true);
        try {
            const { GetList } = await import('../../wailsjs/go/services/AttachmentService');
            const list = await GetList(documentId);
            setFiles(list || []);
        } catch (err: any) {
            console.error(err);
            message.error('Не удалось загрузить файлы');
        }
        setLoading(false);
    };

    useEffect(() => {
        if (documentId) {
            loadFiles();
        }
    }, [documentId]);

    const handleUpload = async (file: File) => {
        setUploading(true);
        try {
            const reader = new FileReader();
            reader.readAsDataURL(file);
            reader.onload = async () => {
                const base64Content = reader.result as string;
                try {
                    const { Upload } = await import('../../wailsjs/go/services/AttachmentService');
                    await Upload(documentId, documentType, file.name, base64Content);
                    message.success('Файл загружен');
                    loadFiles();
                } catch (err: any) {
                    message.error(err?.message || 'Ошибка загрузки');
                } finally {
                    setUploading(false);
                }
            };
            reader.onerror = () => {
                message.error('Ошибка чтения файла');
                setUploading(false);
            };
        } catch (err) {
            setUploading(false);
        }
        return false; // Prevent default upload
    };

    const handleDownload = async (file: any) => {
        try {
            const { DownloadToDisk, OpenFile, OpenFolder } = await import('../../wailsjs/go/services/AttachmentService');

            // Start download
            message.loading({ content: 'Скачивание файла...', key: 'download' });

            const savedPath = await DownloadToDisk(file.id);

            message.success({ content: 'Файл скачан', key: 'download' });

            // Show notification with actions
            const key = `open${Date.now()}`;
            api.open({
                message: 'Скачивание завершено',
                description: `Файл сохранен: ${savedPath}`,
                icon: <FileOutlined style={{ color: '#108ee9' }} />,
                key,
                duration: 0, // Keep open until user interacts
                btn: (
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

        } catch (err: any) {
            console.error(err);
            message.error({ content: err?.message || 'Ошибка скачивания', key: 'download' });
        }
    };

    const handleDelete = async (id: string) => {
        try {
            const { Delete } = await import('../../wailsjs/go/services/AttachmentService');
            await Delete(id);
            message.success('Файл удален');
            loadFiles();
        } catch (err: any) {
            message.error(err?.message || 'Ошибка удаления');
        }
    };

    const getIcon = (filename: string) => {
        const ext = filename.split('.').pop()?.toLowerCase();
        if (ext === 'pdf') return <FilePdfOutlined style={{ color: 'red' }} />;
        if (['doc', 'docx'].includes(ext || '')) return <FileWordOutlined style={{ color: 'blue' }} />;
        if (['jpg', 'png', 'jpeg'].includes(ext || '')) return <FileImageOutlined style={{ color: 'orange' }} />;
        return <FileOutlined />;
    };

    const props: UploadProps = {
        beforeUpload: handleUpload,
        showUploadList: false,
    };

    return (
        <div style={{ padding: 16 }}>
            {contextHolder}
            {canEdit && (
                <div style={{ marginBottom: 16 }}>
                    <Upload {...props}>
                        <Button icon={<UploadOutlined />} loading={uploading}>Загрузить файл</Button>
                    </Upload>
                </div>
            )}

            <List
                loading={loading}
                itemLayout="horizontal"
                dataSource={files}
                renderItem={(item) => (
                    <List.Item
                        actions={[
                            <Tooltip title="Скачать">
                                <Button type="text" icon={<DownloadOutlined />} onClick={() => handleDownload(item)} />
                            </Tooltip>,
                            canEdit && (
                                <Popconfirm title="Удалить файл?" onConfirm={() => handleDelete(item.id)}>
                                    <Button type="text" danger icon={<DeleteOutlined />} />
                                </Popconfirm>
                            )
                        ].filter(Boolean)}
                    >
                        <List.Item.Meta
                            avatar={<Avatar icon={getIcon(item.filename)} style={{ backgroundColor: 'transparent', color: 'inherit' }} />}
                            title={<a onClick={() => handleDownload(item)}>{item.filename}</a>}
                            description={
                                <Text type="secondary" style={{ fontSize: 12 }}>
                                    {(item.fileSize / 1024).toFixed(1)} KB • {item.uploadedByName} • {new Date(item.uploadedAt).toLocaleDateString()}
                                </Text>
                            }
                        />
                    </List.Item>
                )}
            />
        </div>
    );
};

export default FileListComponent;
