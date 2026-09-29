import dayjs from 'dayjs';
import React, { useState } from 'react';
import { Alert, Button, Table } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import AcknowledgmentModal from './AcknowledgmentModal';
import { useAuthStore } from '../store/useAuthStore';
import AssignmentModal from './AssignmentModal';
import { useAssignments } from '../hooks/useAssignments';
import { buildAssignmentColumns } from './assignmentListColumns';
import AssignmentSeriesModal from './AssignmentSeriesModal';

interface AssignmentListProps {
    documentId: string;
    documentKind: string;
}

const AssignmentList: React.FC<AssignmentListProps> = ({ documentId, documentKind }) => {
    const {
        data,
        loadWarning,
        loading,
        initialLoading,
        accessReady,
        canManageAssignments,
        load,
        deleteAssignment,
    } = useAssignments({ documentId, documentKind });
    const { user } = useAuthStore();
    const [acknowledgmentOpen, setAcknowledgmentOpen] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const [editAssignment, setEditAssignment] = useState<any>(null);
    const [seriesAssignment, setSeriesAssignment] = useState<any>(null);

    const columns = buildAssignmentColumns({
        canManageAssignments,
        currentUserId: user?.id,
        onEdit: (assignment) => {
            setEditAssignment(assignment);
            if (assignment.type === 'acknowledgment') setAcknowledgmentOpen(true);
            else setModalOpen(true);
        },
        onDelete: deleteAssignment,
        onManageSeries: setSeriesAssignment,
    });

    const handleAssignmentsChanged = async () => {
        await load();
    };

    return (
        <div>
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
                {canManageAssignments && <Button onClick={() => { setEditAssignment(null); setAcknowledgmentOpen(true); }}>На ознакомление</Button>}
                {canManageAssignments && (
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => { setEditAssignment(null); setModalOpen(true); }}>
                        Добавить поручение
                    </Button>
                )}
            </div>

            {loadWarning && <Alert
                type="warning"
                showIcon
                title={loadWarning}
                action={<Button size="small" loading={loading} onClick={() => void load()}>Обновить</Button>}
            />}
            <Table
                columns={columns}
                dataSource={data}
                rowKey="id"
                size="small"
                pagination={false}
                scroll={{ x: 1000 }}
                loading={initialLoading || !accessReady}
                expandable={{
                    expandedRowRender: (record) => (
                        <div style={{ margin: 0 }}>
                            {record.type === 'acknowledgment' && (record.users || []).map(u => <p key={u.userId}>{u.userName}: {u.confirmedAt ? `ознакомлен ${dayjs(u.confirmedAt).format('DD.MM.YYYY HH:mm')}` : 'ожидает ознакомления'}</p>)}
                            {record.report && (
                                <p><b>{record.status === 'returned' ? 'Причина возврата:' : 'Отчет об исполнении:'}</b> {record.report}</p>
                            )}
                        </div>
                    ),
                    rowExpandable: (record) => !!record.report || record.type === 'acknowledgment',
                }}
            />

            <AcknowledgmentModal open={acknowledgmentOpen} documentId={documentId} initialValues={editAssignment}
                onCancel={() => { setAcknowledgmentOpen(false); setEditAssignment(null); }} onSuccess={handleAssignmentsChanged} />
            <AssignmentModal
                open={modalOpen}
                onCancel={() => { setModalOpen(false); setEditAssignment(null); }}
                onSuccess={handleAssignmentsChanged}
                documentId={documentId}
                isEdit={!!editAssignment}
                initialValues={editAssignment}
            />

            <AssignmentSeriesModal
                open={!!seriesAssignment}
                seriesId={seriesAssignment?.seriesId || ''}
                documentId={documentId}
                onCancel={() => setSeriesAssignment(null)}
                onSuccess={handleAssignmentsChanged}
            />
        </div>
    );
};

export default AssignmentList;
