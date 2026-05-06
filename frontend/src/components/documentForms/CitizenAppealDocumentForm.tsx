import React from 'react';
import { Button, Col, DatePicker, Form, Input, InputNumber, Row, Select, Switch } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import locale from 'antd/es/date-picker/locale/ru_RU';

const { TextArea } = Input;

type Option = {
    value: string;
    label: string;
};

const APPEAL_TYPE_OPTIONS = [
    { value: 'предложение', label: 'Предложение' },
    { value: 'заявление', label: 'Заявление' },
    { value: 'жалоба', label: 'Жалоба' },
];

type CitizenAppealDocumentFormProps = {
    form: any;
    isEdit: boolean;
    onFinish: (values: any) => void;
    nomenclatures: any[];
    orgOptions: Option[];
    executorOptions: Option[];
    onOrgSearch: (query: string) => void;
    onExecutorSearch: (query: string) => void;
};

const smallAddButtonStyle = { height: 24, paddingInline: 8, fontSize: 12 };

const CitizenAppealDocumentForm: React.FC<CitizenAppealDocumentFormProps> = ({
    form,
    isEdit,
    onFinish,
    nomenclatures,
    orgOptions,
    executorOptions,
    onOrgSearch,
    onExecutorSearch,
}) => (
    <Form form={form} layout="vertical" onFinish={onFinish}>
        {!isEdit && (
            <Row gutter={16}>
                <Col span={6}>
                    <Form.Item name="nomenclatureId" label="Дело" rules={[{ required: true, message: 'Выберите дело' }]}>
                        <Select
                            placeholder="Выберите дело"
                            options={nomenclatures.map((n: any) => ({ value: n.id, label: `${n.index} — ${n.name}` }))}
                        />
                    </Form.Item>
                </Col>
                <Col span={6}>
                    <Form.Item name="registrationDate" label="Дата регистрации" rules={[{ required: true, message: 'Укажите дату регистрации' }]}>
                        <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
                    </Form.Item>
                </Col>
                <Col span={6}>
                    <Form.Item name="appealDate" label="Дата обращения" rules={[{ required: true, message: 'Укажите дату обращения' }]}>
                        <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
                    </Form.Item>
                </Col>
                <Col span={6}>
                    <Form.Item name="receivedFromPos" label="ПОС" valuePropName="checked">
                        <Switch checkedChildren="Да" unCheckedChildren="Нет" />
                    </Form.Item>
                </Col>
            </Row>
        )}

        {isEdit && (
            <Row gutter={16}>
                <Col span={8}>
                    <Form.Item name="registrationDate" label="Дата регистрации" rules={[{ required: true, message: 'Укажите дату регистрации' }]}>
                        <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
                    </Form.Item>
                </Col>
                <Col span={8}>
                    <Form.Item name="appealDate" label="Дата обращения" rules={[{ required: true, message: 'Укажите дату обращения' }]}>
                        <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
                    </Form.Item>
                </Col>
                <Col span={8}>
                    <Form.Item name="receivedFromPos" label="ПОС" valuePropName="checked">
                        <Switch checkedChildren="Да" unCheckedChildren="Нет" />
                    </Form.Item>
                </Col>
            </Row>
        )}

        <Row gutter={16}>
            <Col span={12}>
                <Form.Item name="applicantFullName" label="ФИО обратившегося" rules={[{ required: true, whitespace: true, message: 'Укажите ФИО' }]}>
                    <Input />
                </Form.Item>
            </Col>
            <Col span={12}>
                <Form.Item name="applicantCategory" label="Категория обратившегося" rules={[{ required: true, whitespace: true, message: 'Укажите категорию' }]}>
                    <Input />
                </Form.Item>
            </Col>
        </Row>

        <Form.Item name="registrationAddress" label="Адрес регистрации" rules={[{ required: true, whitespace: true, message: 'Укажите адрес регистрации' }]}>
            <Input />
        </Form.Item>

        <Row gutter={16}>
            <Col span={6}>
                <Form.Item name="appealType" label="Вид обращения" rules={[{ required: true, message: 'Выберите вид обращения' }]}>
                    <Select options={APPEAL_TYPE_OPTIONS} />
                </Form.Item>
            </Col>
            <Col span={6}>
                <Form.Item name="appealPagesCount" label="Листов обращения" rules={[{ required: true, message: 'Укажите количество' }]}>
                    <InputNumber min={1} style={{ width: '100%' }} />
                </Form.Item>
            </Col>
            <Col span={6}>
                <Form.Item name="attachmentPagesCount" label="Листов приложения" rules={[{ required: true, message: 'Укажите количество' }]}>
                    <InputNumber min={0} style={{ width: '100%' }} />
                </Form.Item>
            </Col>
            <Col span={6}>
                <Form.Item name="hasEnvelope" label="Конверт" valuePropName="checked">
                    <Switch checkedChildren="Да" unCheckedChildren="Нет" />
                </Form.Item>
            </Col>
        </Row>

        <Form.Item name="content" label="Содержание" rules={[{ required: true, whitespace: true, message: 'Укажите содержание' }]}>
            <TextArea rows={3} />
        </Form.Item>

        <Form.List name="correspondents">
            {(fields, { add, remove }) => (
                <div style={{ marginBottom: 8 }}>
                    {fields.map((field) => {
                        const { key: fieldKey, ...restField } = field;

                        return (
                            <div key={fieldKey} style={{ marginBottom: 12 }}>
                                <Row gutter={12} align="top">
                                    <Col span={7}>
                                        <Form.Item {...restField} name={[field.name, 'registrationNumber']} label="Рег. №" rules={[{ required: true, message: 'Укажите номер' }]}>
                                            <Input />
                                        </Form.Item>
                                    </Col>
                                    <Col span={6}>
                                        <Form.Item {...restField} name={[field.name, 'registrationDate']} label="Дата" rules={[{ required: true, message: 'Укажите дату' }]}>
                                            <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
                                        </Form.Item>
                                    </Col>
                                    <Col span={fields.length > 1 ? 9 : 11}>
                                        <Form.Item {...restField} name={[field.name, 'correspondentName']} label="Корреспондент" rules={[{ required: true, message: 'Укажите корреспондента' }]}>
                                            <Select
                                                showSearch
                                                filterOption={false}
                                                onSearch={onOrgSearch}
                                                options={orgOptions}
                                                notFoundContent={null}
                                                onInputKeyDown={(e) => { if (e.key === ' ') e.stopPropagation(); }}
                                            />
                                        </Form.Item>
                                    </Col>
                                    {fields.length > 1 && (
                                        <Col span={2}>
                                            <Form.Item label=" " colon={false}>
                                                <Button icon={<DeleteOutlined />} onClick={() => remove(field.name)} />
                                            </Form.Item>
                                        </Col>
                                    )}
                                </Row>
                            </div>
                        );
                    })}
                    <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                        <Button type="dashed" size="small" icon={<PlusOutlined />} onClick={() => add()} style={smallAddButtonStyle}>
                            Добавить
                        </Button>
                    </div>
                </div>
            )}
        </Form.List>

        <Form.List name="resolutions">
            {(fields, { add, remove }) => (
                <div>
                    {fields.map((field, index) => {
                        const { key: fieldKey, ...restField } = field;

                        return (
                            <div key={fieldKey} style={{ marginBottom: 12 }}>
                                <Form.Item {...restField} name={[field.name, 'resolution']} label={`Резолюция ${index + 1}`}>
                                    <TextArea rows={2} />
                                </Form.Item>
                                <Row gutter={12}>
                                    <Col span={11}>
                                        <Form.Item {...restField} name={[field.name, 'resolutionAuthor']} label="Автор резолюции">
                                            <Input />
                                        </Form.Item>
                                    </Col>
                                    <Col span={fields.length > 1 ? 11 : 13}>
                                        <Form.Item {...restField} name={[field.name, 'resolutionExecutors']} label="Исполнители резолюции">
                                            <Select
                                                mode="tags"
                                                filterOption={false}
                                                onSearch={onExecutorSearch}
                                                options={executorOptions}
                                                notFoundContent={null}
                                                onInputKeyDown={(e) => { if (e.key === ' ') e.stopPropagation(); }}
                                            />
                                        </Form.Item>
                                    </Col>
                                    {fields.length > 1 && (
                                        <Col span={2}>
                                            <Form.Item label=" " colon={false}>
                                                <Button icon={<DeleteOutlined />} onClick={() => remove(field.name)} />
                                            </Form.Item>
                                        </Col>
                                    )}
                                </Row>
                            </div>
                        );
                    })}
                    <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                        <Button type="dashed" size="small" icon={<PlusOutlined />} onClick={() => add()} style={smallAddButtonStyle}>
                            Добавить резолюцию
                        </Button>
                    </div>
                </div>
            )}
        </Form.List>
    </Form>
);

export default CitizenAppealDocumentForm;
