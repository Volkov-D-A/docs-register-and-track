import React from 'react';
import { Collapse, Tag } from 'antd';
import { FilterOutlined } from '@ant-design/icons';

type DocumentFilterPanelProps = {
    hasFilters: boolean;
    children: React.ReactNode;
};

const DocumentFilterPanel: React.FC<DocumentFilterPanelProps> = ({ hasFilters, children }) => (
    <Collapse
        size="small"
        style={{ marginBottom: 16 }}
        items={[{
            key: 'filters',
            label: (
                <span>
                    <FilterOutlined /> Расширенный поиск
                    {hasFilters ? <Tag color="blue" style={{ marginLeft: 8 }}>Активны</Tag> : null}
                </span>
            ),
            children,
        }]}
    />
);

export default DocumentFilterPanel;
