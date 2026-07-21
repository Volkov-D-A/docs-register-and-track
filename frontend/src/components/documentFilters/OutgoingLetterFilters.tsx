import React from 'react';
import { Col, Input, Row } from 'antd';
import { ClearFiltersButton, DateRangeFilter, FilterFieldLabel } from './filterPrimitives';

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
                <FilterFieldLabel label="Исх. номер">
                    <Input size="small" value={filterOutgoingNumber} onChange={e => onOutgoingNumberChange(e.target.value)} placeholder="Исх. номер" allowClear />
                </FilterFieldLabel>
            </Col>
            <Col span={8}>
                <FilterFieldLabel label="Получатель">
                    <Input size="small" value={filterRecipientName} onChange={e => onRecipientNameChange(e.target.value)} placeholder="Организация" allowClear />
                </FilterFieldLabel>
            </Col>
            <Col span={8}>
                <DateRangeFilter label="Дата (диапазон)" from={filterDateFrom} to={filterDateTo} onChange={onDateRangeChange} />
            </Col>
        </Row>
        {hasFilters && (
            <div style={{ marginTop: 8, display: 'flex', justifyContent: 'flex-end' }}>
                <ClearFiltersButton visible={hasFilters} onClick={onClear} />
            </div>
        )}
    </div>
);

export default OutgoingLetterFilters;
