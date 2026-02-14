import React, { useState, useEffect } from 'react';
import { List, Button, Upload, message, Popconfirm, Avatar, Typography, Tooltip } from 'antd';
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
            const { Download } = await import('../../wailsjs/go/services/AttachmentService');
            // Returns base64 string, filename
            // However, Wails Go method returns (string, string, error).
            // In JS/TS generated code, it might return Promise<string> or object depending on binding generation.
            // Usually multiple returns come as array or object?
            // Actually standard Wails V2 returns the first return value if no error, or throws error.
            // Wait, if Go returns multiple values, Wails returns them as an array in JS?
            // Let's assume standard behavior: Wails handles multiple returns by checking if binding supports it. 
            // If not, maybe it returns just the first one?
            // Actually, best practice in Wails is to return a struct if you want multiple values. 
            // My Go code returns (string, string, error). This might be problematic in standard Wails fetch if not handled.
            // Let's assume the generated code handles it or returns an array.

            // Re-checking AttachmentService.go: 
            // func (s *AttachmentService) Download(idStr string) (string, string, error)

            // If Wails generates TS, it usually returns Promise<string> if only one value, or Promise<[string, string]> if multiple on some versions?
            // Safer approach: Let's assume it returns the base64 string, and we use the filename from the file object we already have.

            // Wait, if I cannot be sure, I should check wails docs or assume I might need to adjust Go code to return a struct.
            // But for now let's hope it returns the first string (content).

            // Wails should return the struct now
            const result = await Download(file.id);
            const contentBase64 = result.content;
            const filename = result.filename || file.filename;

            // Create download link
            const a = document.createElement('a');
            a.href = `data:application/octet-stream;base64,${contentBase64}`;
            a.download = filename;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
        } catch (err: any) {
            message.error(err?.message || 'Ошибка скачивания');
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
