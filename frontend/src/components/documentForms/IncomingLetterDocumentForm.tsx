import React from 'react';
import { Button, Col, DatePicker, Form, Input, InputNumber, Row, Select, Tooltip } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import locale from 'antd/es/date-picker/locale/ru_RU';

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
                    <Form.Item name="nomenclatureId" label="Дело" rules={[{ required: true }]}>
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
                        <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
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
        <Form.List name="correspondents">
            {(fields, { add, remove }) => (
                <div style={{ marginBottom: 8 }}>
                    {fields.map((field, index) => {
                        const { key: fieldKey, ...restField } = field;

                        return (
                            <div
                                key={fieldKey}
                                style={{
                                    marginBottom: 12,
                                }}
                            >
                                <Row gutter={12} align="top">
                                    <Col span={7}>
                                        <Form.Item
                                            {...restField}
                                            name={[field.name, 'registrationNumber']}
                                            label="Регистрационный номер"
                                            rules={[{ required: true, message: 'Укажите номер' }]}
                                        >
                                            <Input />
                                        </Form.Item>
                                    </Col>
                                    <Col span={6}>
                                        <Form.Item
                                            {...restField}
                                            name={[field.name, 'registrationDate']}
                                            label="Дата"
                                            rules={[{ required: true, message: 'Укажите дату' }]}
                                        >
                                            <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
                                        </Form.Item>
                                    </Col>
                                    <Col span={fields.length > 1 ? 9 : 11}>
                                        <Form.Item
                                            {...restField}
                                            name={[field.name, 'correspondentName']}
                                            label="Корреспондент"
                                            rules={[{ required: true, message: 'Укажите корреспондента' }]}
                                        >
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
                                    {fields.length > 1 && (
                                        <Col span={2}>
                                            <Form.Item label={index === 0 ? ' ' : ' '} colon={false}>
                                                <Tooltip title="Удалить корреспондента">
                                                    <Button icon={<DeleteOutlined />} onClick={() => remove(field.name)} />
                                                </Tooltip>
                                            </Form.Item>
                                        </Col>
                                    )}
                                </Row>
                            </div>
                        );
                    })}
                    <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                        <Button type="dashed" size="small" icon={<PlusOutlined />} onClick={() => add()} style={{ height: 24, paddingInline: 8, fontSize: 12 }}>
                            Добавить
                        </Button>
                    </div>
                </div>
            )}
        </Form.List>
        <Row gutter={16}>
            <Col span={12}>
                <Form.Item name="senderSignatory" label="Подписант" rules={[{ required: true, message: 'Укажите подписанта' }]}>
                    <Input />
                </Form.Item>
            </Col>
            <Col span={12}>
                <Form.Item name="pagesCount" label="Кол-во листов" rules={[{ required: true, message: 'Укажите кол-во' }]}>
                    <InputNumber min={1} style={{ width: '100%' }} />
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
