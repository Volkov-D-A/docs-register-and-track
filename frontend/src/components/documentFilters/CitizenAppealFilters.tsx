import React from 'react';
import { Col, Input, Row, Select } from 'antd';
import { ClearFiltersButton, DateRangeFilter, FilterFieldLabel } from './filterPrimitives';

const appealTypeOptions = [
    { value: 'предложение', label: 'Предложение' },
    { value: 'заявление', label: 'Заявление' },
    { value: 'жалоба', label: 'Жалоба' },
];

type CitizenAppealFiltersProps = {
    hasFilters: boolean;
    filterRegistrationNumber: string;
    filterApplicantName: string;
    filterAppealType: string;
    filterRegistrationDateFrom: string;
    filterRegistrationDateTo: string;
    filterAppealDateFrom: string;
    filterAppealDateTo: string;
    filterResolution: string;
    filterNoResolution: boolean;
    onRegistrationNumberChange: (value: string) => void;
    onApplicantNameChange: (value: string) => void;
    onAppealTypeChange: (value: string) => void;
    onRegistrationDateRangeChange: (from: string, to: string) => void;
    onAppealDateRangeChange: (from: string, to: string) => void;
    onResolutionChange: (value: string) => void;
    onNoResolutionChange: (value: boolean) => void;
    onClear: () => void;
};

const CitizenAppealFilters: React.FC<CitizenAppealFiltersProps> = ({
    hasFilters,
    filterRegistrationNumber,
    filterApplicantName,
    filterAppealType,
    filterRegistrationDateFrom,
    filterRegistrationDateTo,
    filterAppealDateFrom,
    filterAppealDateTo,
    filterResolution,
    filterNoResolution,
    onRegistrationNumberChange,
    onApplicantNameChange,
    onAppealTypeChange,
    onRegistrationDateRangeChange,
    onAppealDateRangeChange,
    onResolutionChange,
    onNoResolutionChange,
    onClear,
}) => (
    <div>
        <Row gutter={16}>
            <Col span={6}>
                <FilterFieldLabel label="Номер документа">
                    <Input size="small" value={filterRegistrationNumber} onChange={e => onRegistrationNumberChange(e.target.value)} allowClear />
                </FilterFieldLabel>
            </Col>
            <Col span={6}>
                <FilterFieldLabel label="ФИО обратившегося">
                    <Input size="small" value={filterApplicantName} onChange={e => onApplicantNameChange(e.target.value)} allowClear />
                </FilterFieldLabel>
            </Col>
            <Col span={6}>
                <FilterFieldLabel label="Вид обращения">
                    <Select
                        size="small"
                        value={filterAppealType || undefined}
                        onChange={onAppealTypeChange}
                        allowClear
                        options={appealTypeOptions}
                        style={{ width: '100%' }}
                    />
                </FilterFieldLabel>
            </Col>
            <Col span={6} style={{ display: 'flex', alignItems: 'flex-end', paddingBottom: 8 }}>
                <ClearFiltersButton visible={hasFilters} onClick={onClear} />
            </Col>
        </Row>
        <Row gutter={16}>
            <Col span={12}>
                <DateRangeFilter label="Дата регистрации" from={filterRegistrationDateFrom} to={filterRegistrationDateTo} onChange={onRegistrationDateRangeChange} />
            </Col>
            <Col span={12}>
                <DateRangeFilter label="Дата обращения" from={filterAppealDateFrom} to={filterAppealDateTo} onChange={onAppealDateRangeChange} />
            </Col>
        </Row>
        <Row gutter={16}>
            <Col span={8}>
                <FilterFieldLabel label="Резолюция">
                    <Input
                        size="small"
                        value={filterResolution}
                        onChange={e => onResolutionChange(e.target.value)}
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

export default CitizenAppealFilters;
