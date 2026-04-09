import React, { useEffect, useState } from 'react';
import { Table, Button, Tag, Space, Popconfirm, Modal, Input, Tooltip, App } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, CheckCircleOutlined, PlayCircleOutlined, CloseCircleOutlined, UndoOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useAuthStore } from '../store/useAuthStore';
import AssignmentModal from './AssignmentModal';
import AssignmentCompletionModal from './AssignmentCompletionModal';

/**
 * Свойства компонента AssignmentList.
 */
interface AssignmentListProps {
    documentId: string;
}

const { TextArea } = Input;

/**
 * Компонент списка поручений по документу.
 * @param documentId Идентификатор документа
 * @param documentType Тип документа
 */
const AssignmentList: React.FC<AssignmentListProps> = ({ documentId }) => {
    const { message } = App.useApp();
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const [editAssignment, setEditAssignment] = useState<any>(null);
    const { user, hasRole, currentRole } = useAuthStore();
    const isExecutorOnly = currentRole === 'executor';

    // Report modal
    const [completionModalOpen, setCompletionModalOpen] = useState(false);
    const [currentAssignment, setCurrentAssignment] = useState<any>(null);
    const [returnModalOpen, setReturnModalOpen] = useState(false);
    const [returnReasonText, setReturnReasonText] = useState('');

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

    const handleReturnToRevision = () => {
        if (!returnReasonText.trim()) {
            message.error('Введите причину возврата');
            return;
        }
        updateStatus(currentAssignment.id, 'returned', returnReasonText);
        setReturnModalOpen(false);
        setReturnReasonText('');
        setCurrentAssignment(null);
    };

    const columns = [
        {
            title: 'Дата', dataIndex: 'createdAt', key: 'createdAt', width: 100,
            render: (v: string) => dayjs(v).format('DD.MM.YYYY'),
        },
        { title: 'Содержание', dataIndex: 'content', key: 'content' },
        {
            title: 'Исполнитель', key: 'executorName', width: 200,
            render: (_: any, r: any) => (
                <div>
                    <div>{r.executorName}</div>
                    {r.coExecutors && r.coExecutors.length > 0 && (
                        <div style={{ fontSize: '11px', color: '#888' }}>
                            + {r.coExecutors.map((u: any) => u.fullName).join(', ')}
                        </div>
                    )}
                </div>
            )
        },
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
                const isClerk = hasRole('clerk');

                const canEdit = isClerk && r.status !== 'finished';

                return (
                    <Space size={2}>
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
                                    onClick={() => { setCurrentAssignment(r); setCompletionModalOpen(true); }} />
                            </Tooltip>
                        )}

                        {isClerk && r.status === 'completed' && (
                            <Tooltip title="Вернуть на доработку">
                                <Button
                                    size="small"
                                    icon={<UndoOutlined />}
                                    onClick={() => {
                                        setCurrentAssignment(r);
                                        setReturnReasonText('');
                                        setReturnModalOpen(true);
                                    }}
                                />
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
                                <p><b>{record.status === 'returned' ? 'Причина возврата:' : 'Отчет об исполнении:'}</b> {record.report}</p>
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
                isEdit={!!editAssignment}
                initialValues={editAssignment}
            />

            <AssignmentCompletionModal
                open={completionModalOpen}
                assignmentId={currentAssignment?.id || ''}
                documentId={currentAssignment?.documentId || ''}
                initialReport={currentAssignment?.report || ''}
                onCancel={() => {
                    setCompletionModalOpen(false);
                    setCurrentAssignment(null);
                }}
                onSuccess={() => {
                    setCompletionModalOpen(false);
                    setCurrentAssignment(null);
                    load();
                }}
            />

            <Modal
                title="Причина возврата на доработку"
                open={returnModalOpen}
                onCancel={() => {
                    setReturnModalOpen(false);
                    setReturnReasonText('');
                    setCurrentAssignment(null);
                }}
                onOk={handleReturnToRevision}
                okText="Вернуть"
            >
                <TextArea
                    rows={4}
                    value={returnReasonText}
                    onChange={e => setReturnReasonText(e.target.value)}
                    placeholder="Введите причину возврата..."
                />
            </Modal>
        </div>
    );
};

export default AssignmentList;
