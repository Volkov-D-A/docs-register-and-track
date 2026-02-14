import React, { useEffect, useState } from 'react';
import { Table, Button, Tag, Space, Popconfirm, message, Modal, Input, Tooltip } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, CheckCircleOutlined, PlayCircleOutlined, CloseCircleOutlined, UndoOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useAuthStore } from '../store/useAuthStore';
import AssignmentModal from './AssignmentModal';

interface AssignmentListProps {
    documentId: string;
    documentType: 'incoming' | 'outgoing';
}

const { TextArea } = Input;

const AssignmentList: React.FC<AssignmentListProps> = ({ documentId, documentType }) => {
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const [editAssignment, setEditAssignment] = useState<any>(null);
    const { user, hasRole, currentRole } = useAuthStore();
    const isExecutorOnly = currentRole === 'executor';

    // Report modal
    const [reportModalOpen, setReportModalOpen] = useState(false);
    const [currentAssignmentId, setCurrentAssignmentId] = useState<string>('');
    const [reportText, setReportText] = useState('');

    const load = async () => {
        if (!documentId) return;
        setLoading(true);
        try {
            const { GetList } = await import('../../wailsjs/go/services/AssignmentService');
            const result = await GetList({ documentId, page: 1, pageSize: 100, showFinished: true, overdueOnly: false });
            setData(result?.items || []);
        } catch (err) {
            console.error(err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { load(); }, [documentId]);

    const onDelete = async (id: string) => {
        try {
            const { Delete } = await import('../../wailsjs/go/services/AssignmentService');
            await Delete(id);
            message.success('Удалено');
            load();
        } catch (err: any) {
            message.error(err?.message || String(err));
        }
    };

    const updateStatus = async (id: string, status: string, report: string = '') => {
        try {
            const { UpdateStatus } = await import('../../wailsjs/go/services/AssignmentService');
            await UpdateStatus(id, status, report);
            message.success('Статус обновлен');
            load();
        } catch (err: any) {
            message.error(err?.message || String(err));
        }
    };

    const handleComplete = () => {
        if (!reportText.trim()) {
            message.error('Введите отчет об исполнении');
            return;
        }
        updateStatus(currentAssignmentId, 'completed', reportText);
        setReportModalOpen(false);
        setReportText('');
    };

    const columns = [
        {
            title: 'Дата', dataIndex: 'createdAt', key: 'createdAt', width: 100,
            render: (v: string) => dayjs(v).format('DD.MM.YYYY'),
        },
        { title: 'Содержание', dataIndex: 'content', key: 'content' },
        { title: 'Исполнитель', dataIndex: 'executorName', key: 'executorName', width: 150 },
        {
            title: 'Срок', dataIndex: 'deadline', key: 'deadline', width: 100,
            render: (v: string) => v ? dayjs(v).format('DD.MM.YYYY') : '',
        },
        {
            title: 'Статус', dataIndex: 'status', key: 'status', width: 120,
            render: (status: string, record: any) => {
                let color = 'default';
                let text = status;

                // Check for overdue completion (only for completed status)
                // If completedAt is after deadline (day granularity)
                const isOverdue = status === 'completed' && record.completedAt && record.deadline &&
                    dayjs(record.completedAt).isAfter(dayjs(record.deadline), 'day');

                switch (status) {
                    case 'new': color = 'blue'; text = 'Новое'; break;
                    case 'in_progress': color = 'orange'; text = 'В работе'; break;
                    case 'completed':
                        if (isOverdue) {
                            color = 'red';
                            text = 'Исполнено (просрочено)';
                        } else {
                            color = 'green';
                            text = 'Исполнено';
                        }
                        break;
                    case 'finished': color = 'geekblue'; text = 'Завершен'; break;
                    case 'cancelled': color = 'red'; text = 'Отменено'; break;
                    case 'returned': color = 'volcano'; text = 'Возврат'; break;
                }
                return <Tag color={color}>{text}</Tag>;
            }
        },
        {
            title: '', key: 'actions', width: 150,
            render: (_: any, r: any) => {
                const isExecutor = user?.id === r.executorId && currentRole === 'executor';
                const isAdmin = hasRole('admin');
                const canEdit = isAdmin;

                return (
                    <Space size={2}>
                        {/* Edit/Delete for Admin */}
                        {canEdit && (
                            <>
                                <Button size="small" icon={<EditOutlined />} onClick={() => { setEditAssignment(r); setModalOpen(true); }} />
                                <Popconfirm title="Удалить поручение?" onConfirm={() => onDelete(r.id)}>
                                    <Button size="small" icon={<DeleteOutlined />} danger />
                                </Popconfirm>
                            </>
                        )}

                        {/* Status Actions */}
                        {/* Start: Executor, status=new/returned */}
                        {isExecutor && (r.status === 'new' || r.status === 'returned') && (
                            <Tooltip title="Взять в работу">
                                <Button size="small" icon={<PlayCircleOutlined />} onClick={() => updateStatus(r.id, 'in_progress')} />
                            </Tooltip>
                        )}

                        {/* Complete: Executor, status=in_progress */}
                        {isExecutor && r.status === 'in_progress' && (
                            <Tooltip title="Исполнить">
                                <Button size="small" icon={<CheckCircleOutlined />}
                                    onClick={() => { setCurrentAssignmentId(r.id); setReportText(r.report || ''); setReportModalOpen(true); }} />
                            </Tooltip>
                        )}

                        {/* Return/Cancel: Admin, status=completed/in_progress */}
                        {isAdmin && r.status === 'completed' && (
                            <Tooltip title="Вернуть на доработку">
                                <Button size="small" icon={<UndoOutlined />} onClick={() => updateStatus(r.id, 'returned')} />
                            </Tooltip>
                        )}

                        {isAdmin && r.status !== 'cancelled' && r.status !== 'completed' && (
                            <Tooltip title="Отменить">
                                <Button size="small" icon={<CloseCircleOutlined />} danger onClick={() => updateStatus(r.id, 'cancelled')} />
                            </Tooltip>
                        )}
                    </Space>
                );
            }
        },
    ];

    return (
        <div>
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end' }}>
                {!isExecutorOnly && (
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => { setEditAssignment(null); setModalOpen(true); }}>
                        Добавить поручение
                    </Button>
                )}
            </div>

            <Table
                columns={columns}
                dataSource={data}
                rowKey="id"
                size="small"
                pagination={false}
                loading={loading}
                expandable={{
                    expandedRowRender: (record) => (
                        <div style={{ margin: 0 }}>
                            {record.report && (
                                <p><b>Отчет об исполнении:</b> {record.report}</p>
                            )}
                        </div>
                    ),
                    rowExpandable: (record) => !!record.report,
                }}
            />

            <AssignmentModal
                open={modalOpen}
                onCancel={() => { setModalOpen(false); setEditAssignment(null); }}
                onSuccess={load}
                documentId={documentId}
                documentType={documentType}
                isEdit={!!editAssignment}
                initialValues={editAssignment}
            />

            <Modal
                title="Отчет об исполнении"
                open={reportModalOpen}
                onCancel={() => setReportModalOpen(false)}
                onOk={handleComplete}
                okText="Исполнено"
            >
                <TextArea
                    rows={4}
                    value={reportText}
                    onChange={e => setReportText(e.target.value)}
                    placeholder="Введите результат выполнения поручения..."
                />
            </Modal>
        </div>
    );
};

export default AssignmentList;
