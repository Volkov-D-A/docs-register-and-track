import React, { useState } from 'react';
import { App, Button, Input, Modal, Table } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import AssignmentModal from './AssignmentModal';
import AssignmentCompletionModal from './AssignmentCompletionModal';
import { useAssignments } from '../hooks/useAssignments';
import { buildAssignmentColumns } from './assignmentListColumns';

interface AssignmentListProps {
    documentId: string;
    documentKind: string;
}

const { TextArea } = Input;

const AssignmentList: React.FC<AssignmentListProps> = ({ documentId, documentKind }) => {
    const { message } = App.useApp();
    const { user } = useAuthStore();
    const {
        data,
        loading,
        accessReady,
        canManageAssignments,
        load,
        deleteAssignment,
        updateStatus,
    } = useAssignments({ documentId, documentKind });
    const [modalOpen, setModalOpen] = useState(false);
    const [editAssignment, setEditAssignment] = useState<any>(null);
    const [completionModalOpen, setCompletionModalOpen] = useState(false);
    const [currentAssignment, setCurrentAssignment] = useState<any>(null);
    const [returnModalOpen, setReturnModalOpen] = useState(false);
    const [returnReasonText, setReturnReasonText] = useState('');

    const handleReturnToRevision = () => {
        if (!returnReasonText.trim()) {
            message.error('Введите причину возврата');
            return;
        }
        void updateStatus(currentAssignment.id, 'returned', returnReasonText);
        setReturnModalOpen(false);
        setReturnReasonText('');
        setCurrentAssignment(null);
    };

    const columns = buildAssignmentColumns({
        userId: user?.id,
        canManageAssignments,
        onEdit: (assignment) => {
            setEditAssignment(assignment);
            setModalOpen(true);
        },
        onDelete: deleteAssignment,
        onUpdateStatus: updateStatus,
        onOpenCompletion: (assignment) => {
            setCurrentAssignment(assignment);
            setCompletionModalOpen(true);
        },
        onOpenReturn: (assignment) => {
            setCurrentAssignment(assignment);
            setReturnReasonText('');
            setReturnModalOpen(true);
        },
    });

    return (
        <div>
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end' }}>
                {canManageAssignments && (
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
                loading={loading || !accessReady}
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
                    void load();
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

export default AssignmentList;
