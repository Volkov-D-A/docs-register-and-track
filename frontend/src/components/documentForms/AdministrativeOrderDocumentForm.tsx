import React from 'react';
import { Button, Col, DatePicker, Form, Input, Row, Select, Switch } from 'antd';
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';
import locale from 'antd/es/date-picker/locale/ru_RU';

const { TextArea } = Input;

type AdministrativeOrderDocumentFormProps = {
    form: any;
    isEdit: boolean;
    onFinish: (values: any) => void;
    nomenclatures: any[];
    selectedRegisterNomenclature?: any;
};

const AdministrativeOrderDocumentForm: React.FC<AdministrativeOrderDocumentFormProps> = ({
    form,
    isEdit,
    onFinish,
    nomenclatures,
    selectedRegisterNomenclature,
}) => {
    const isActive = Form.useWatch('isActive', form);

    return (
        <Form form={form} layout="vertical" onFinish={onFinish}>
            {!isEdit && (
                <Row gutter={16}>
                    <Col span={12}>
                        <Form.Item name="nomenclatureId" label="Дело" rules={[{ required: true, message: 'Выберите дело' }]}>
                            <Select options={nomenclatures.map((n: any) => ({ value: n.id, label: `${n.index} — ${n.name}` }))} placeholder="Выберите дело" />
                        </Form.Item>
                    </Col>
                    <Col span={12}>
                        <Form.Item name="orderDate" label="Дата приказа" rules={[{ required: true, message: 'Укажите дату приказа' }]}>
                            <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
                        </Form.Item>
                    </Col>
                </Row>
            )}

            {isEdit && (
                <Form.Item name="orderDate" label="Дата приказа" rules={[{ required: true, message: 'Укажите дату приказа' }]}>
                    <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
                </Form.Item>
            )}

            {!isEdit && selectedRegisterNomenclature?.numberingMode === 'manual_only' && (
                <Form.Item name="registrationNumber" label="Регистрационный номер" rules={[{ required: true, message: 'Введите номер вручную' }]}>
                    <Input placeholder="Введите номер приказа" />
                </Form.Item>
            )}

            <Form.Item name="title" label="Заголовок" rules={[{ required: true, message: 'Введите заголовок' }]}>
                <TextArea rows={3} />
            </Form.Item>

            <Row gutter={16}>
                <Col span={12}>
                    <Form.Item name="executionController" label="Контроль за выполнением">
                        <Input placeholder="ФИО контролирующего" />
                    </Form.Item>
                </Col>
                <Col span={12}>
                    <Form.Item name="executionDeadline" label="Срок выполнения">
                        <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
                    </Form.Item>
                </Col>
            </Row>

            <Row gutter={16}>
                <Col span={12}>
                    <Form.Item name="isActive" label="Статус приказа" valuePropName="checked">
                        <Switch checkedChildren="Действующий" unCheckedChildren="Не действующий" />
                    </Form.Item>
                </Col>
                {!isActive && (
                    <Col span={12}>
                        <Form.Item name="cancelledAt" label="Дата отмены" rules={[{ required: true, message: 'Укажите дату отмены' }]}>
                            <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
                        </Form.Item>
                    </Col>
                )}
            </Row>

            <Form.List name="acknowledgmentFullNames">
                {(fields, { add, remove }) => (
                    <div>
                        <div style={{ marginBottom: 8, fontWeight: 500 }}>Ознакомить</div>
                        {fields.map((field) => (
                            <Row key={field.key} gutter={8} align="middle">
                                <Col flex="auto">
                                    <Form.Item {...field} rules={[{ required: true, message: 'Введите ФИО' }]}>
                                        <Input placeholder="ФИО" />
                                    </Form.Item>
                                </Col>
                                <Col flex="32px">
                                    <Button type="text" danger icon={<MinusCircleOutlined />} onClick={() => remove(field.name)} />
                                </Col>
                            </Row>
                        ))}
                        <Button type="dashed" icon={<PlusOutlined />} onClick={() => add()} block>
                            Добавить ФИО
                        </Button>
                    </div>
                )}
            </Form.List>
        </Form>
    );
};

export default AdministrativeOrderDocumentForm;
