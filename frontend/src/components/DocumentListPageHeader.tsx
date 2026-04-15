import React from 'react';
import { Button, Input, Space, Typography } from 'antd';
import { PlusOutlined, SearchOutlined } from '@ant-design/icons';

const { Title } = Typography;

type DocumentListPageHeaderProps = {
    title: string;
    nomenclatureFilter?: React.ReactNode;
    onSearch: (value: string) => void;
    canRegister: boolean;
    onRegister: () => void;
};

const DocumentListPageHeader: React.FC<DocumentListPageHeaderProps> = ({
    title,
    nomenclatureFilter,
    onSearch,
    canRegister,
    onRegister,
}) => (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>{title}</Title>
        <Space>
            {nomenclatureFilter}
            <Input.Search
                placeholder="Поиск по содержанию..."
                allowClear
                onSearch={onSearch}
                style={{ width: 250 }}
                prefix={<SearchOutlined />}
            />
            {canRegister && (
                <Button type="primary" icon={<PlusOutlined />} onClick={onRegister}>
                    Зарегистрировать
                </Button>
            )}
        </Space>
    </div>
);

export default DocumentListPageHeader;
