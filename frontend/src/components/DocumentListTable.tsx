import React from 'react';
import { Table } from 'antd';

type DocumentListTableProps = {
    className: string;
    columns: any[];
    data: any[];
    loading: boolean;
    page: number;
    pageSize: number;
    totalCount: number;
    onPageChange: (page: number, pageSize: number) => void;
};

const DocumentListTable: React.FC<DocumentListTableProps> = ({
    className,
    columns,
    data,
    loading,
    page,
    pageSize,
    totalCount,
    onPageChange,
}) => (
    <Table
        className={className}
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        size="small"
        tableLayout="fixed"
        pagination={{
            current: page,
            pageSize,
            total: totalCount,
            onChange: onPageChange,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50'],
        }}
    />
);

export default DocumentListTable;
