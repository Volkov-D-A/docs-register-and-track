import React from 'react';
import { Col, Input, Row } from 'antd';
import { ClearFiltersButton, DateRangeFilter, FilterFieldLabel } from './filterPrimitives';

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
                <FilterFieldLabel label="Вх. номер">
                    <Input size="small" value={filterIncomingNumber} onChange={e => onIncomingNumberChange(e.target.value)} placeholder="Рег. номер" allowClear />
                </FilterFieldLabel>
            </Col>
            <Col span={6}>
                <FilterFieldLabel label="Регистрационный номер корреспондента">
                    <Input size="small" value={filterOutgoingNumber} onChange={e => onOutgoingNumberChange(e.target.value)} placeholder="Номер у корреспондента" allowClear />
                </FilterFieldLabel>
            </Col>
            <Col span={6}>
                <FilterFieldLabel label="Корреспондент">
                    <Input size="small" value={filterSenderName} onChange={e => onSenderNameChange(e.target.value)} placeholder="Название организации" allowClear />
                </FilterFieldLabel>
            </Col>
            <Col span={6} style={{ display: 'flex', alignItems: 'flex-end', paddingBottom: 8 }}>
                <ClearFiltersButton visible={hasFilters} onClick={onClear} />
            </Col>
        </Row>
        <Row gutter={16}>
            <Col span={12}>
                <DateRangeFilter label="Дата получения (диапазон)" from={filterDateFrom} to={filterDateTo} onChange={onDateRangeChange} />
            </Col>
            <Col span={12}>
                <DateRangeFilter label="Дата корреспондента (диапазон)" from={filterOutDateFrom} to={filterOutDateTo} onChange={onOutgoingDateRangeChange} />
            </Col>
        </Row>
        <Row gutter={16}>
            <Col span={8}>
                <FilterFieldLabel label="Резолюция">
                    <Input
                        size="small"
                        value={filterResolution}
                        onChange={e => onResolutionChange(e.target.value)}
                        placeholder="Текст резолюции"
                        allowClear
                        disabled={filterNoResolution}
                    />
                </FilterFieldLabel>
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
