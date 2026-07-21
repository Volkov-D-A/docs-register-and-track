import React from 'react';
import { Col, DatePicker, Form, Input, Row, Select } from 'antd';
import locale from 'antd/es/date-picker/locale/ru_RU';
import { DocumentContentField, ManualRegistrationNumberField, PagesCountField } from './formBlocks';

type Option = {
    value: string;
    label: string;
};

type OutgoingLetterDocumentFormProps = {
    form: any;
    isEdit: boolean;
    onFinish: (values: any) => void;
    nomenclatures: any[];
    docTypes: any[];
    orgOptionsRecipient: Option[];
    selectedRegisterNomenclature?: any;
    onRecipientOrgSearch: (query: string) => void;
};

const OutgoingLetterDocumentForm: React.FC<OutgoingLetterDocumentFormProps> = ({
    form,
    isEdit,
    onFinish,
    nomenclatures,
    docTypes,
    orgOptionsRecipient,
    selectedRegisterNomenclature,
    onRecipientOrgSearch,
}) => (
    <Form form={form} layout="vertical" onFinish={onFinish}>
        {!isEdit && (
            <>
                <Row gutter={16}>
                    <Col span={12}>
                        <Form.Item name="nomenclatureId" label="Дело" rules={[{ required: true, message: 'Выберите дело' }]}>
                            <Select options={nomenclatures.map((n: any) => ({ value: n.id, label: `${n.index} — ${n.name}` }))} placeholder="Выберите дело" />
                        </Form.Item>
                    </Col>
                    <Col span={12}>
                        <Form.Item name="outgoingDate" label="Исходящая дата" rules={[{ required: true }]}>
                            <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
                        </Form.Item>
                    </Col>
                </Row>
                <Row gutter={16}>
                    <Col span={12}>
                        <Form.Item name="documentTypeId" label="Тип документа" rules={[{ required: true }]}>
                            <Select options={docTypes.map((t: any) => ({ value: t.id, label: t.name }))} />
                        </Form.Item>
                    </Col>
                    <Col span={12}>
                        <PagesCountField />
                    </Col>
                </Row>
                {selectedRegisterNomenclature?.numberingMode === 'manual_only' && (
                    <ManualRegistrationNumberField />
                )}
            </>
        )}
        {isEdit && (
            <>
                <Row gutter={16}>
                    <Col span={12}>
                        <Form.Item name="outgoingDate" label="Исходящая дата" rules={[{ required: true }]}>
                            <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
                        </Form.Item>
                    </Col>
                    <Col span={12}>
                        <PagesCountField />
                    </Col>
                </Row>
                <Row gutter={16}>
                    <Col span={12}>
                        <Form.Item name="documentTypeId" label="Тип документа" rules={[{ required: true }]}>
                            <Select options={docTypes.map((t: any) => ({ value: t.id, label: t.name }))} />
                        </Form.Item>
                    </Col>
                </Row>
            </>
        )}

        <Row gutter={16}>
            <Col span={12}>
                <Form.Item name="recipientOrgName" label="Получатель (Организация)" rules={[{ required: true }]}>
                    <Select
                        showSearch
                        onSearch={onRecipientOrgSearch}
                        options={orgOptionsRecipient}
                        notFoundContent={null}
                        onInputKeyDown={(e) => { if (e.key === ' ' && !e.isDefaultPrevented()) e.stopPropagation(); }}
                    />
                </Form.Item>
            </Col>
            <Col span={12}>
                <Form.Item name="addressee" label="Адресат (ФИО)" rules={[{ required: true }]}>
                    <Input />
                </Form.Item>
            </Col>
        </Row>

        <Row gutter={16}>
            <Col span={12}>
                <Form.Item name="senderSignatory" label="Кто подписывает" rules={[{ required: true }]}>
                    <Input />
                </Form.Item>
            </Col>
            <Col span={12}>
                <Form.Item name="senderExecutor" label="Исполнитель письма" rules={[{ required: true }]}>
                    <Input />
                </Form.Item>
            </Col>
        </Row>

        <DocumentContentField rows={4} />
    </Form>
);

export default OutgoingLetterDocumentForm;
