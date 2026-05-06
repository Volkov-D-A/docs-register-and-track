import React from 'react';
import { Button, Checkbox, Col, DatePicker, Input, Row, Select, Typography } from 'antd';
import { ClearOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

const { Text } = Typography;
const { RangePicker } = DatePicker;

type AdministrativeOrderFiltersProps = {
    hasFilters: boolean;
    filterOrderNumber: string;
    filterExecutionController: string;
    filterDateFrom: string;
    filterDateTo: string;
    filterOnlyControlled: boolean;
    filterOnlyOverdue: boolean;
    filterOnlyPendingAcknowledgment: boolean;
    filterOrderActiveStatus: string;
    onOrderNumberChange: (value: string) => void;
    onExecutionControllerChange: (value: string) => void;
    onDateRangeChange: (from: string, to: string) => void;
    onOnlyControlledChange: (value: boolean) => void;
    onOnlyOverdueChange: (value: boolean) => void;
    onOnlyPendingAcknowledgmentChange: (value: boolean) => void;
    onOrderActiveStatusChange: (value: string) => void;
    onClear: () => void;
};

const AdministrativeOrderFilters: React.FC<AdministrativeOrderFiltersProps> = ({
    hasFilters,
    filterOrderNumber,
    filterExecutionController,
    filterDateFrom,
    filterDateTo,
    filterOnlyControlled,
    filterOnlyOverdue,
    filterOnlyPendingAcknowledgment,
    filterOrderActiveStatus,
    onOrderNumberChange,
    onExecutionControllerChange,
    onDateRangeChange,
    onOnlyControlledChange,
    onOnlyOverdueChange,
    onOnlyPendingAcknowledgmentChange,
    onOrderActiveStatusChange,
    onClear,
}) => (
    <div>
        <Row gutter={16}>
            <Col span={6}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Номер</Text>
                    <Input size="small" value={filterOrderNumber} onChange={e => onOrderNumberChange(e.target.value)} placeholder="Номер приказа" allowClear />
                </div>
            </Col>
            <Col span={6}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Контроль</Text>
                    <Input size="small" value={filterExecutionController} onChange={e => onExecutionControllerChange(e.target.value)} placeholder="ФИО" allowClear />
                </div>
            </Col>
            <Col span={6}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Дата</Text>
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
            <Col span={6}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Статус</Text>
                    <Select
                        size="small"
                        style={{ width: '100%' }}
                        value={filterOrderActiveStatus}
                        onChange={onOrderActiveStatusChange}
                        options={[
                            { value: '', label: 'Все' },
                            { value: 'active', label: 'Действующие' },
                            { value: 'inactive', label: 'Не действующие' },
                        ]}
                    />
                </div>
            </Col>
        </Row>
        <Row gutter={16}>
            <Col span={8}>
                <Checkbox checked={filterOnlyControlled} onChange={e => onOnlyControlledChange(e.target.checked)}>На контроле</Checkbox>
            </Col>
            <Col span={8}>
                <Checkbox checked={filterOnlyOverdue} onChange={e => onOnlyOverdueChange(e.target.checked)}>Просрочен срок</Checkbox>
            </Col>
            <Col span={8}>
                <Checkbox checked={filterOnlyPendingAcknowledgment} onChange={e => onOnlyPendingAcknowledgmentChange(e.target.checked)}>Незавершенное ознакомление</Checkbox>
            </Col>
        </Row>
        {hasFilters && (
            <div style={{ marginTop: 8, display: 'flex', justifyContent: 'flex-end' }}>
                <Button size="small" icon={<ClearOutlined />} onClick={onClear}>Сбросить фильтры</Button>
            </div>
        )}
    </div>
);

export default AdministrativeOrderFilters;
