import React, { useMemo, useState } from 'react';
import { App, Button, Input, Modal, Space, Spin, Tag, Tooltip, Typography } from 'antd';
import { CheckCircleOutlined, FileDoneOutlined, PlayCircleOutlined, UndoOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import AssignmentCompletionModal from './AssignmentCompletionModal';
import { useAssignments } from '../hooks/useAssignments';
import { useAuthStore } from '../store/useAuthStore';

interface DocumentAssignmentWorkflowPanelProps {
    documentId: string;
    documentKind: string;
    onAssignmentsChanged?: () => void | Promise<void>;
}

const { Text } = Typography;
const { TextArea } = Input;

const getStatusTag = (status: string) => {
    switch (status) {
        case 'new':
            return <Tag color="blue">Новое</Tag>;
        case 'in_progress':
            return <Tag color="orange">В работе</Tag>;
        case 'completed':
            return <Tag color="green">Исполнено</Tag>;
        case 'finished':
            return <Tag color="geekblue">Завершён</Tag>;
        case 'returned':
            return <Tag color="volcano">Возврат</Tag>;
        case 'cancelled':
            return <Tag color="red">Отменено</Tag>;
        default:
            return <Tag>{status}</Tag>;
    }
};

const DocumentAssignmentWorkflowPanel: React.FC<DocumentAssignmentWorkflowPanelProps> = ({
    documentId,
    documentKind,
    onAssignmentsChanged,
}) => {
    const { message } = App.useApp();
    const { user } = useAuthStore();
    const {
        data,
        loading,
        accessReady,
        canManageAssignments,
        load,
        updateStatus,
    } = useAssignments({ documentId, documentKind });
    const [completionModalOpen, setCompletionModalOpen] = useState(false);
    const [returnModalOpen, setReturnModalOpen] = useState(false);
    const [currentAssignment, setCurrentAssignment] = useState<any>(null);
    const [returnReasonText, setReturnReasonText] = useState('');

    const actionableAssignments = useMemo(() => data.filter((assignment) => {
        const isExecutor = user?.id === assignment.executorId;
        const executorCanAct = isExecutor && ['new', 'returned', 'in_progress'].includes(assignment.status);
        const managerCanAct = canManageAssignments && assignment.status === 'completed';
        return executorCanAct || managerCanAct;
    }), [canManageAssignments, data, user?.id]);

    const notifyAssignmentsChanged = async () => {
        await onAssignmentsChanged?.();
    };

    const handleUpdateStatus = async (id: string, status: string, report = '') => {
        const updated = await updateStatus(id, status, report);
        if (updated) {
            await notifyAssignmentsChanged();
        }
    };

    if (!loading && accessReady && actionableAssignments.length === 0) {
        return null;
    }

    const handleReturnToRevision = () => {
        if (!returnReasonText.trim()) {
            message.error('Введите причину возврата');
            return;
        }
        void handleUpdateStatus(currentAssignment.id, 'returned', returnReasonText);
        setReturnModalOpen(false);
        setReturnReasonText('');
        setCurrentAssignment(null);
    };

    return (
        <div className="document-assignment-workflow">
            <div className="document-assignment-workflow__header">
                <Text strong>Поручения к исполнению</Text>
                {loading && <Spin size="small" />}
            </div>

            {!loading && actionableAssignments.map((assignment) => {
                const isExecutor = user?.id === assignment.executorId;
                return (
                    <div className="document-assignment-workflow__item" key={assignment.id}>
                        <div className="document-assignment-workflow__body">
                            <div className="document-assignment-workflow__content">{assignment.content}</div>
                            <Space size={4} wrap>
                                {getStatusTag(assignment.status)}
                                {assignment.executorName && <Text type="secondary">{assignment.executorName}</Text>}
                                {assignment.deadline && (
                                    <Text type="secondary">Срок: {dayjs(assignment.deadline).format('DD.MM.YYYY')}</Text>
                                )}
                            </Space>
                            {assignment.report && (
                                <div className="document-assignment-workflow__report">
                                    <Text type="secondary">
                                        <Text strong type="secondary">
                                            {assignment.status === 'returned' ? 'Причина возврата:' : 'Отчет об исполнении:'}
                                        </Text>{' '}
                                        {assignment.report}
                                    </Text>
                                </div>
                            )}
                        </div>
                        <Space size={6} wrap className="document-assignment-workflow__actions">
                            {isExecutor && (assignment.status === 'new' || assignment.status === 'returned') && (
                                <Tooltip title="Взять в работу">
                                    <Button
                                        size="small"
                                        icon={<PlayCircleOutlined />}
                                        onClick={() => handleUpdateStatus(assignment.id, 'in_progress')}
                                    >
                                        Взять в работу
                                    </Button>
                                </Tooltip>
                            )}
                            {isExecutor && assignment.status === 'in_progress' && (
                                <Tooltip title="Исполнить">
                                    <Button
                                        size="small"
                                        type="primary"
                                        icon={<CheckCircleOutlined />}
                                        onClick={() => {
                                            setCurrentAssignment(assignment);
                                            setCompletionModalOpen(true);
                                        }}
                                    >
                                        Исполнить
                                    </Button>
                                </Tooltip>
                            )}
                            {canManageAssignments && assignment.status === 'completed' && (
                                <>
                                    <Tooltip title="Завершить">
                                        <Button
                                            size="small"
                                            type="primary"
                                            icon={<FileDoneOutlined />}
                                            onClick={() => handleUpdateStatus(assignment.id, 'finished')}
                                        >
                                            Завершить
                                        </Button>
                                    </Tooltip>
                                    <Tooltip title="Вернуть на доработку">
                                        <Button
                                            size="small"
                                            icon={<UndoOutlined />}
                                            onClick={() => {
                                                setCurrentAssignment(assignment);
                                                setReturnReasonText('');
                                                setReturnModalOpen(true);
                                            }}
                                        >
                                            Вернуть
                                        </Button>
                                    </Tooltip>
                                </>
                            )}
                        </Space>
                    </div>
                );
            })}

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
                    void load();
                    void notifyAssignmentsChanged();
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
                    onChange={(event) => setReturnReasonText(event.target.value)}
                    placeholder="Введите причину возврата..."
                />
            </Modal>
        </div>
    );
};

export default DocumentAssignmentWorkflowPanel;
