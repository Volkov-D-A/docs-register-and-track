import React from 'react';
import { Button, Col, DatePicker, Input, Row, Typography } from 'antd';
import { ClearOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

const { Text } = Typography;
const { RangePicker } = DatePicker;

type OutgoingLetterFiltersProps = {
    hasFilters: boolean;
    filterOutgoingNumber: string;
    filterRecipientName: string;
    filterDateFrom: string;
    filterDateTo: string;
    onOutgoingNumberChange: (value: string) => void;
    onRecipientNameChange: (value: string) => void;
    onDateRangeChange: (from: string, to: string) => void;
    onClear: () => void;
};

const OutgoingLetterFilters: React.FC<OutgoingLetterFiltersProps> = ({
    hasFilters,
    filterOutgoingNumber,
    filterRecipientName,
    filterDateFrom,
    filterDateTo,
    onOutgoingNumberChange,
    onRecipientNameChange,
    onDateRangeChange,
    onClear,
}) => (
    <div>
        <Row gutter={16}>
            <Col span={8}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Исх. номер</Text>
                    <Input size="small" value={filterOutgoingNumber} onChange={e => onOutgoingNumberChange(e.target.value)} placeholder="Исх. номер" allowClear />
                </div>
            </Col>
            <Col span={8}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Получатель</Text>
                    <Input size="small" value={filterRecipientName} onChange={e => onRecipientNameChange(e.target.value)} placeholder="Организация" allowClear />
                </div>
            </Col>
            <Col span={8}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Дата (диапазон)</Text>
                    <RangePicker
                        size="small"
                        style={{ width: '100%' }}
                        format="DD.MM.YYYY"
                        value={filterDateFrom && filterDateTo ? [dayjs(filterDateFrom), dayjs(filterDateTo)] : null}
                        onChange={(dates) => onDateRangeChange(
                            dates?.[0]?.format('YYYY-MM-DD') || '',
                            dates?.[1]?.format('YYYY-MM-DD') || '',
                        )}
                    />
                </div>
            </Col>
        </Row>
        {hasFilters && (
            <div style={{ marginTop: 8, display: 'flex', justifyContent: 'flex-end' }}>
                <Button size="small" icon={<ClearOutlined />} onClick={onClear}>Сбросить фильтры</Button>
            </div>
        )}
    </div>
);

export default OutgoingLetterFilters;
