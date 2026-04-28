import React from 'react';
import { Button, Col, DatePicker, Input, Row, Select, Typography } from 'antd';
import { ClearOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

const { Text } = Typography;
const { RangePicker } = DatePicker;

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
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Номер документа</Text>
                    <Input size="small" value={filterRegistrationNumber} onChange={e => onRegistrationNumberChange(e.target.value)} allowClear />
                </div>
            </Col>
            <Col span={6}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>ФИО обратившегося</Text>
                    <Input size="small" value={filterApplicantName} onChange={e => onApplicantNameChange(e.target.value)} allowClear />
                </div>
            </Col>
            <Col span={6}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Вид обращения</Text>
                    <Select
                        size="small"
                        value={filterAppealType || undefined}
                        onChange={onAppealTypeChange}
                        allowClear
                        options={appealTypeOptions}
                        style={{ width: '100%' }}
                    />
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
                    <Text type="secondary" style={{ fontSize: 12 }}>Дата регистрации</Text>
                    <RangePicker
                        size="small"
                        style={{ width: '100%' }}
                        format="DD.MM.YYYY"
                        value={filterRegistrationDateFrom && filterRegistrationDateTo ? [dayjs(filterRegistrationDateFrom), dayjs(filterRegistrationDateTo)] : null}
                        onChange={(dates) => onRegistrationDateRangeChange(
                            dates?.[0]?.format('YYYY-MM-DD') || '',
                            dates?.[1]?.format('YYYY-MM-DD') || '',
                        )}
                    />
                </div>
            </Col>
            <Col span={12}>
                <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Дата обращения</Text>
                    <RangePicker
                        size="small"
                        style={{ width: '100%' }}
                        format="DD.MM.YYYY"
                        value={filterAppealDateFrom && filterAppealDateTo ? [dayjs(filterAppealDateFrom), dayjs(filterAppealDateTo)] : null}
                        onChange={(dates) => onAppealDateRangeChange(
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

export default CitizenAppealFilters;
