import React from 'react';
import { Button, Col, DatePicker, Input, Row, Typography } from 'antd';
import { ClearOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

const { Text } = Typography;
const { RangePicker } = DatePicker;

type IncomingLetterFiltersProps = {
    hasFilters: boolean;
    filterIncomingNumber: string;
    filterOutgoingNumber: string;
    filterSenderName: string;
    filterDateFrom: string;
    filterDateTo: string;
    filterOutDateFrom: string;
    filterOutDateTo: string;
    filterResolution: string;
    filterNoResolution: boolean;
    onIncomingNumberChange: (value: string) => void;
    onOutgoingNumberChange: (value: string) => void;
    onSenderNameChange: (value: string) => void;
    onDateRangeChange: (from: string, to: string) => void;
    onOutgoingDateRangeChange: (from: string, to: string) => void;
    onResolutionChange: (value: string) => void;
    onNoResolutionChange: (value: boolean) => void;
    onClear: () => void;
};

const IncomingLetterFilters: React.FC<IncomingLetterFiltersProps> = ({
    hasFilters,
    filterIncomingNumber,
    filterOutgoingNumber,
    filterSenderName,
    filterDateFrom,
    filterDateTo,
    filterOutDateFrom,
    filterOutDateTo,
    filterResolution,
    filterNoResolution,
    onIncomingNumberChange,
    onOutgoingNumberChange,
    onSenderNameChange,
    onDateRangeChange,
    onOutgoingDateRangeChange,
    onResolutionChange,
    onNoResolutionChange,
    onClear,
}) => (
    <div>
        <Row gutter={16}>
            <Col span={6}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Вх. номер</Text>
                    <Input size="small" value={filterIncomingNumber} onChange={e => onIncomingNumberChange(e.target.value)} placeholder="Рег. номер" allowClear />
                </div>
            </Col>
            <Col span={6}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Исх. номер</Text>
                    <Input size="small" value={filterOutgoingNumber} onChange={e => onOutgoingNumberChange(e.target.value)} placeholder="Исх. номер" allowClear />
                </div>
            </Col>
            <Col span={6}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Отправитель</Text>
                    <Input size="small" value={filterSenderName} onChange={e => onSenderNameChange(e.target.value)} placeholder="Название организации" allowClear />
                </div>
            </Col>
            <Col span={6} style={{ display: 'flex', alignItems: 'flex-end', paddingBottom: 8 }}>
                {hasFilters && (
                    <Button size="small" icon={<ClearOutlined />} onClick={onClear}>Сбросить фильтры</Button>
                )}
            </Col>
        </Row>
        <Row gutter={16}>
            <Col span={12}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Дата получения (диапазон)</Text>
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
            <Col span={12}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Дата отправки (диапазон)</Text>
                    <RangePicker
                        size="small"
                        style={{ width: '100%' }}
                        format="DD.MM.YYYY"
                        value={filterOutDateFrom && filterOutDateTo ? [dayjs(filterOutDateFrom), dayjs(filterOutDateTo)] : null}
                        onChange={(dates) => onOutgoingDateRangeChange(
                            dates?.[0]?.format('YYYY-MM-DD') || '',
                            dates?.[1]?.format('YYYY-MM-DD') || '',
                        )}
                    />
                </div>
            </Col>
        </Row>
        <Row gutter={16}>
            <Col span={8}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Резолюция</Text>
                    <Input
                        size="small"
                        value={filterResolution}
                        onChange={e => onResolutionChange(e.target.value)}
                        placeholder="Текст резолюции"
                        allowClear
                        disabled={filterNoResolution}
                    />
                </div>
            </Col>
            <Col span={8} style={{ display: 'flex', alignItems: 'flex-end', paddingBottom: 8 }}>
                <label style={{ fontSize: 12, cursor: 'pointer' }}>
                    <input
                        type="checkbox"
                        checked={filterNoResolution}
                        onChange={e => onNoResolutionChange(e.target.checked)}
                        style={{ marginRight: 6 }}
                    />
                    Без резолюции
                </label>
            </Col>
        </Row>
    </div>
);

export default IncomingLetterFilters;
