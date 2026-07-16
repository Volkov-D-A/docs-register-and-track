import React from 'react';
import { Button, Empty, Popconfirm, Space, Spin, Tooltip, Typography } from 'antd';
import { DeleteOutlined, DownloadOutlined, FileImageOutlined, FileOutlined, FilePdfOutlined, FileWordOutlined, UploadOutlined } from '@ant-design/icons';
import { useAttachments } from '../hooks/useAttachments';

const { Text } = Typography;

interface FileListComponentProps {
    documentId: string;
    documentKind: string;
    readOnly?: boolean;
}

const getIcon = (filename: string) => {
    const ext = filename.split('.').pop()?.toLowerCase();
    if (ext === 'pdf') return <FilePdfOutlined style={{ color: 'red' }} />;
    if (['doc', 'docx'].includes(ext || '')) return <FileWordOutlined style={{ color: 'blue' }} />;
    if (['jpg', 'png', 'jpeg'].includes(ext || '')) return <FileImageOutlined style={{ color: 'orange' }} />;
    return <FileOutlined />;
};

const FileListComponent: React.FC<FileListComponentProps> = ({ documentId, documentKind, readOnly }) => {
    const {
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
    } = useAttachments({ documentId, documentKind, readOnly });

    return (
        <div style={{ padding: 16 }}>
            {contextHolder}
            {canEdit && (
                <div style={{ marginBottom: 16 }}>
                    <Button icon={<UploadOutlined />} loading={uploading} onClick={() => void uploadFile()}>Загрузить файл</Button>
                </div>
            )}

            {loading || !accessReady ? (
                <div style={{ textAlign: 'center', padding: 20 }}>
                    <Spin />
                </div>
            ) : files.length === 0 ? (
                <Empty
                    description={
                        <span>
                            Файлы не прикреплены. {canEdit ? 'Нажмите "Загрузить файл", чтобы добавить вложение.' : 'Обратитесь к пользователю с правом редактирования, если вложение нужно добавить.'}
                        </span>
                    }
                />
            ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                    {files.map((item) => (
                        <div key={item.id} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: 12, border: '1px solid var(--app-border)', borderRadius: 8 }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 12, flex: 1, minWidth: 0 }}>
                                <div style={{ fontSize: 24, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                    {getIcon(item.filename)}
                                </div>
                                <div style={{ minWidth: 0 }}>
                                    <div style={{ marginBottom: 4 }}>
                                        <a onClick={() => downloadFile(item)} style={{ color: '#1677ff', fontWeight: 500, wordBreak: 'break-word' }}>
                                            {item.filename}
                                        </a>
                                    </div>
                                    <Text type="secondary" style={{ fontSize: 12 }}>
                                        {(item.fileSize / 1024).toFixed(1)} KB • {item.uploadedByName} • {new Date(item.uploadedAt).toLocaleDateString()}
                                    </Text>
                                </div>
                            </div>
                            <Space style={{ flexShrink: 0, marginLeft: 16 }}>
                                <Tooltip title="Скачать">
                                    <Button type="text" icon={<DownloadOutlined />} onClick={() => downloadFile(item)} />
                                </Tooltip>
                                {canEdit && (
                                    <Popconfirm
                                        title={`Удалить файл "${item.filename}"?`}
                                        description="Это действие нельзя отменить. Файл будет удалён из вложений документа."
                                        okText="Удалить"
                                        cancelText="Отмена"
                                        okButtonProps={{ danger: true, loading: deletingIds.has(item.id) }}
                                        onConfirm={() => deleteFile(item.id)}
                                    >
                                        <Button
                                            type="text"
                                            title="Удалить файл"
                                            danger
                                            icon={<DeleteOutlined />}
                                            loading={deletingIds.has(item.id)}
                                        />
                                    </Popconfirm>
                                )}
                            </Space>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
};

export default FileListComponent;
