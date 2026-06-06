import React, { useState } from 'react';
import { Button, Table } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import AssignmentModal from './AssignmentModal';
import { useAssignments } from '../hooks/useAssignments';
import { buildAssignmentColumns } from './assignmentListColumns';

interface AssignmentListProps {
    documentId: string;
    documentKind: string;
}

const AssignmentList: React.FC<AssignmentListProps> = ({ documentId, documentKind }) => {
    const {
        data,
        loading,
        accessReady,
        canManageAssignments,
        load,
        deleteAssignment,
    } = useAssignments({ documentId, documentKind });
    const [modalOpen, setModalOpen] = useState(false);
    const [editAssignment, setEditAssignment] = useState<any>(null);

    const columns = buildAssignmentColumns({
        canManageAssignments,
        onEdit: (assignment) => {
            setEditAssignment(assignment);
            setModalOpen(true);
        },
        onDelete: deleteAssignment,
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
        </div>
    );
};

export default AssignmentList;
