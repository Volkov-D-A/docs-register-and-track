import React from 'react';
import { Button, DatePicker, Typography } from 'antd';
import { ClearOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

const { Text } = Typography;
const { RangePicker } = DatePicker;

export const FilterFieldLabel: React.FC<React.PropsWithChildren<{ label: string }>> = ({ label, children }) => (
    <div style={{ marginBottom: 8 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>{label}</Text>
        {children}
    </div>
);

export const DateRangeFilter = ({ label, from, to, onChange }: {
    label: string;
    from: string;
    to: string;
    onChange: (from: string, to: string) => void;
}) => <FilterFieldLabel label={label}>
    <RangePicker
        size="small"
        style={{ width: '100%', marginTop: 4 }}
        format="DD.MM.YYYY"
        value={from && to ? [dayjs(from), dayjs(to)] : null}
        onChange={(dates) => onChange(dates?.[0]?.format('YYYY-MM-DD') || '', dates?.[1]?.format('YYYY-MM-DD') || '')}
    />
</FilterFieldLabel>;

export const ClearFiltersButton = ({ visible, onClick }: { visible: boolean; onClick: () => void }) => (
    visible ? <Button size="small" icon={<ClearOutlined />} onClick={onClick}>Сбросить фильтры</Button> : null
);
