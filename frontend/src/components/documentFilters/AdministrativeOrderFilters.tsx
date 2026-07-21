import React from 'react';
import { Checkbox, Col, Input, Row, Select } from 'antd';
import { ClearFiltersButton, DateRangeFilter, FilterFieldLabel } from './filterPrimitives';

type AdministrativeOrderFiltersProps = {
    hasFilters: boolean;
    filterOrderNumber: string;
    filterExecutionController: string;
    filterDateFrom: string;
    filterDateTo: string;
    filterOnlyPendingAcknowledgment: boolean;
    filterOrderActiveStatus: string;
    onOrderNumberChange: (value: string) => void;
    onExecutionControllerChange: (value: string) => void;
    onDateRangeChange: (from: string, to: string) => void;
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
    filterOnlyPendingAcknowledgment,
    filterOrderActiveStatus,
    onOrderNumberChange,
    onExecutionControllerChange,
    onDateRangeChange,
    onOnlyPendingAcknowledgmentChange,
    onOrderActiveStatusChange,
    onClear,
}) => (
    <div>
        <Row gutter={16}>
            <Col span={6}>
                <FilterFieldLabel label="Номер">
                    <Input size="small" value={filterOrderNumber} onChange={e => onOrderNumberChange(e.target.value)} placeholder="Номер приказа" allowClear />
                </FilterFieldLabel>
            </Col>
            <Col span={6}>
                <FilterFieldLabel label="Контроль">
                    <Input size="small" value={filterExecutionController} onChange={e => onExecutionControllerChange(e.target.value)} placeholder="ФИО" allowClear />
                </FilterFieldLabel>
            </Col>
            <Col span={6}>
                <DateRangeFilter label="Дата" from={filterDateFrom} to={filterDateTo} onChange={onDateRangeChange} />
            </Col>
            <Col span={6}>
                <FilterFieldLabel label="Статус">
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
                </FilterFieldLabel>
            </Col>
        </Row>
        <Row gutter={16}>
            <Col span={8}>
                <Checkbox checked={filterOnlyPendingAcknowledgment} onChange={e => onOnlyPendingAcknowledgmentChange(e.target.checked)}>Незавершённое ознакомление</Checkbox>
            </Col>
        </Row>
        {hasFilters && (
            <div style={{ marginTop: 8, display: 'flex', justifyContent: 'flex-end' }}>
                <ClearFiltersButton visible={hasFilters} onClick={onClear} />
            </div>
        )}
    </div>
);

export default AdministrativeOrderFilters;
