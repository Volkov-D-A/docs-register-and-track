import React from 'react';
import { Col, DatePicker, Form, Input, InputNumber, Row, Select } from 'antd';

const { TextArea } = Input;

type Option = {
    value: string;
    label: string;
};

type IncomingLetterDocumentFormProps = {
    form: any;
    isEdit: boolean;
    onFinish: (values: any) => void;
    nomenclatures: any[];
    docTypes: any[];
    selectedRegisterNomenclature?: any;
    orgOptionsSender: Option[];
    executorOptions: Option[];
    onSenderOrgSearch: (query: string) => void;
    onExecutorSearch: (query: string) => void;
};

const IncomingLetterDocumentForm: React.FC<IncomingLetterDocumentFormProps> = ({
    form,
    isEdit,
    onFinish,
    nomenclatures,
    docTypes,
    selectedRegisterNomenclature,
    orgOptionsSender,
    executorOptions,
    onSenderOrgSearch,
    onExecutorSearch,
}) => (
    <Form form={form} layout="vertical" onFinish={onFinish}>
        {!isEdit && (
            <Row gutter={16}>
                <Col span={8}>
                    <Form.Item name="nomenclatureId" label="Дело (номенклатура)" rules={[{ required: true }]}>
                        <Select placeholder="Выберите дело">
                            {nomenclatures.map((n: any) => (
                                <Select.Option key={n.id} value={n.id}>{n.index} — {n.name}</Select.Option>
                            ))}
                        </Select>
                    </Form.Item>
                </Col>
                <Col span={8}>
                    <Form.Item name="documentTypeId" label="Тип документа" rules={[{ required: true }]}>
                        <Select placeholder="Выберите тип">
                            {docTypes.map((t: any) => (
                                <Select.Option key={t.id} value={t.id}>{t.name}</Select.Option>
                            ))}
                        </Select>
                    </Form.Item>
                </Col>
                <Col span={8}>
                    <Form.Item name="incomingDate" label="Дата регистрации" rules={[{ required: true }]}>
                        <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" />
                    </Form.Item>
                </Col>
            </Row>
        )}
        {!isEdit && selectedRegisterNomenclature?.numberingMode === 'manual_only' && (
            <Form.Item name="registrationNumber" label="Регистрационный номер" rules={[{ required: true, message: 'Введите номер вручную' }]}>
                <Input placeholder="Введите номер документа" />
            </Form.Item>
        )}
        {isEdit && (
            <Form.Item name="documentTypeId" label="Тип документа" rules={[{ required: true }]}>
                <Select placeholder="Выберите тип">
                    {docTypes.map((t: any) => (
                        <Select.Option key={t.id} value={t.id}>{t.name}</Select.Option>
                    ))}
                </Select>
            </Form.Item>
        )}
        <Row gutter={16}>
            <Col span={8}>
                <Form.Item name="outgoingNumberSender" label="Исх. № отправителя" rules={[{ required: true, message: 'Укажите исх. номер' }]}>
                    <Input />
                </Form.Item>
            </Col>
            <Col span={8}>
                <Form.Item name="outgoingDateSender" label="Дата исходящего" rules={[{ required: true, message: 'Укажите дату' }]}>
                    <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" />
                </Form.Item>
            </Col>
            <Col span={8}>
                <Form.Item name="pagesCount" label="Кол-во листов" rules={[{ required: true, message: 'Укажите кол-во' }]}>
                    <InputNumber min={1} style={{ width: '100%' }} />
                </Form.Item>
            </Col>
        </Row>
        <Row gutter={16}>
            <Col span={12}>
                <Form.Item name="intermediateNumber" label="Промежуточный номер">
                    <Input placeholder="Необязательно" />
                </Form.Item>
            </Col>
            <Col span={12}>
                <Form.Item name="intermediateDate" label="Промежуточная дата">
                    <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" />
                </Form.Item>
            </Col>
        </Row>
        <Row gutter={16}>
            <Col span={12}>
                <Form.Item name="senderOrgName" label="Организация-отправитель" rules={[{ required: true }]}>
                    <Select
                        showSearch
                        filterOption={false}
                        onSearch={onSenderOrgSearch}
                        options={orgOptionsSender}
                        notFoundContent={null}
                        onInputKeyDown={(e) => { if (e.key === ' ') e.stopPropagation(); }}
                    />
                </Form.Item>
            </Col>
            <Col span={12}>
                <Form.Item name="senderSignatory" label="Подписант" rules={[{ required: true, message: 'Укажите подписанта' }]}>
                    <Input />
                </Form.Item>
            </Col>
        </Row>
        <Form.Item name="content" label="Содержание" rules={[{ required: true }]}>
            <TextArea rows={3} />
        </Form.Item>
        <Form.Item name="resolution" label="Резолюция">
            <TextArea rows={2} placeholder="Текст резолюции" />
        </Form.Item>
        <Row gutter={16}>
            <Col span={12}>
                <Form.Item name="resolutionAuthor" label="Автор резолюции">
                    <Input placeholder="Необязательно" />
                </Form.Item>
            </Col>
            <Col span={12}>
                <Form.Item name="resolutionExecutors" label="Исполнители резолюции">
                    <Select
                        mode="tags"
                        placeholder="Начните вводить ФИО"
                        filterOption={false}
                        onSearch={onExecutorSearch}
                        options={executorOptions}
                        notFoundContent={null}
                        onInputKeyDown={(e) => { if (e.key === ' ') e.stopPropagation(); }}
                    />
                </Form.Item>
            </Col>
        </Row>
    </Form>
);

export default IncomingLetterDocumentForm;
