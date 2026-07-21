import React from 'react';
import { Col, DatePicker, Form, Input, Row, Select, Switch, Tag } from 'antd';
import locale from 'antd/es/date-picker/locale/ru_RU';
import dayjs from 'dayjs';
import { ManualRegistrationNumberField } from './formBlocks';

const { TextArea } = Input;

type AdministrativeOrderDocumentFormProps = {
    form: any;
    isEdit: boolean;
    onFinish: (values: any) => void;
    nomenclatures: any[];
    selectedRegisterNomenclature?: any;
    acknowledgmentPeople?: any[];
};

const AdministrativeOrderDocumentForm: React.FC<AdministrativeOrderDocumentFormProps> = ({
    form,
    isEdit,
    onFinish,
    nomenclatures,
    selectedRegisterNomenclature,
    acknowledgmentPeople = [],
}) => {
    const isActive = Form.useWatch('isActive', form);
    const lockedAcknowledgmentPeople = acknowledgmentPeople.filter((person: any) => !!person.acknowledgedAt);
    const lockedByName = new Map(
        lockedAcknowledgmentPeople.map((person: any) => [person.fullName.trim().toLowerCase(), person])
    );
    const normalizeName = (value: string) => value.trim().toLowerCase();
    const withLockedNames = (values: string[] = []) => {
        const next = [...values];
        const names = new Set(next.map(normalizeName));

        lockedAcknowledgmentPeople.forEach((person: any) => {
            if (!names.has(normalizeName(person.fullName))) {
                next.push(person.fullName);
            }
        });

        return next;
    };

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
                <ManualRegistrationNumberField placeholder="Введите номер приказа" />
            )}

            <Form.Item name="title" label="Заголовок" rules={[{ required: true, message: 'Введите заголовок' }]}>
                <TextArea rows={3} />
            </Form.Item>

            <Row gutter={16}>
                <Col span={12}>
                    <Form.Item
                        name="executionController"
                        label="Контроль за выполнением"
                        rules={[{ required: true, whitespace: true, message: 'Укажите контроль за выполнением' }]}
                    >
                        <Input placeholder="ФИО контролирующего" />
                    </Form.Item>
                </Col>
                <Col span={12}>
                    <Form.Item name="executionDeadline" label="Срок выполнения (справочно)">
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

            <Form.Item name="acknowledgmentFullNames" label="Внешнее ознакомление" normalize={withLockedNames}>
                <Select
                    mode="tags"
                    placeholder="Введите ФИО и нажмите Enter"
                    options={acknowledgmentPeople.map((person: any) => ({ value: person.fullName, label: person.fullName }))}
                    tagRender={({ label, value, closable, onClose }) => {
                        const person = lockedByName.get(normalizeName(String(value)));
                        const isLocked = !!person;
                        const title = isLocked
                            ? `Ознакомлён: ${dayjs(person.acknowledgedAt).format('DD.MM.YYYY HH:mm')}${person.acknowledgedByName ? `, ${person.acknowledgedByName}` : ''}`
                            : undefined;

                        return (
                            <Tag
                                color={isLocked ? 'green' : undefined}
                                closable={!isLocked && closable}
                                onClose={onClose}
                                title={title}
                                style={{ marginInlineEnd: 4 }}
                            >
                                {label}
                            </Tag>
                        );
                    }}
                />
            </Form.Item>
        </Form>
    );
};

export default AdministrativeOrderDocumentForm;
