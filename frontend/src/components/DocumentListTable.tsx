import React from 'react';
import { Button, Space, Table, Typography } from 'antd';

type DocumentListTableProps = {
    className: string;
    columns: any[];
    data: any[];
    loading: boolean;
    page: number;
    pageSize: number;
    hasMore: boolean;
    canGoBack: boolean;
    onPreviousPage: () => void;
    onNextPage: () => void;
    onPageSizeChange: (pageSize: number) => void;
};

const DocumentListTable: React.FC<DocumentListTableProps> = ({
    className,
    columns,
    data,
    loading,
    page,
    pageSize,
    hasMore,
    canGoBack,
    onPreviousPage,
    onNextPage,
    onPageSizeChange,
}) => (
    <Table
        className={className}
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        size="small"
        tableLayout="fixed"
        pagination={false}
        footer={() => (
            <Space style={{ display: 'flex', justifyContent: 'space-between' }}>
                <Typography.Text>Страница {page}</Typography.Text>
                <Space>
                    <Typography.Text type="secondary">По {pageSize}</Typography.Text>
                    {[10, 20, 50].map((size) => (
                        <Button key={size} type={size === pageSize ? 'primary' : 'default'} size="small" onClick={() => onPageSizeChange(size)}>{size}</Button>
                    ))}
                    <Button disabled={!canGoBack || loading} onClick={onPreviousPage}>Назад</Button>
                    <Button type="primary" disabled={!hasMore || loading} onClick={onNextPage}>Вперёд</Button>
                </Space>
            </Space>
        )}
    />
);

export default DocumentListTable;
