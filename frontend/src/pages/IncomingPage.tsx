import React, { useState, useEffect } from 'react';
import {
    Typography, Table, Button, Modal, Form, Input, Select, DatePicker,
    InputNumber, Space, Row, Col, Tag, Popconfirm, AutoComplete, Collapse, Tabs, App,
} from 'antd';
import AssignmentList from '../components/AssignmentList';
import AcknowledgmentList from '../components/AcknowledgmentList';
import FileListComponent from '../components/FileListComponent';
import { LinksTab } from '../components/DocumentLinks/LinksTab';
import DocumentViewModal from '../components/DocumentViewModal';

import {
    PlusOutlined, SearchOutlined, EyeOutlined, DeleteOutlined, EditOutlined,
    FilterOutlined, ClearOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';

import { useAuthStore } from '../store/useAuthStore';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { RangePicker } = DatePicker;

/**
 * Страница входящих документов.
 * Обеспечивает отображение списка, фильтрацию, регистрацию и редактирование входящей корреспонденции.
 */
const IncomingPage: React.FC = () => {
    const { message } = App.useApp();
    const { user, currentRole } = useAuthStore();
    const isExecutorOnly = currentRole === 'executor';
    // Скрываем фильтр, если пользователь — исполнитель без админских прав
    const filterDisabled = isExecutorOnly;

    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [totalCount, setTotalCount] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(10);
    const [search, setSearch] = useState('');

    // Фильтры
    const [filterIncomingNumber, setFilterIncomingNumber] = useState('');
    const [filterOutgoingNumber, setFilterOutgoingNumber] = useState('');
    const [filterSenderName, setFilterSenderName] = useState('');
    const [filterDateFrom, setFilterDateFrom] = useState('');
    const [filterDateTo, setFilterDateTo] = useState('');
    const [filterOutDateFrom, setFilterOutDateFrom] = useState('');
    const [filterOutDateTo, setFilterOutDateTo] = useState('');
    const [filterResolution, setFilterResolution] = useState('');
    const [filterNoResolution, setFilterNoResolution] = useState(false);
    const [filterNomenclatureIds, setFilterNomenclatureIds] = useState<string[]>([]);

    // Модалки
    const [registerModalOpen, setRegisterModalOpen] = useState(false);
    const [editModalOpen, setEditModalOpen] = useState(false);
    const [viewDocId, setViewDocId] = useState<string>('');
    const [viewModalOpen, setViewModalOpen] = useState(false);
    const [editDoc, setEditDoc] = useState<any>(null);

    const [registerForm] = Form.useForm();
    const [editForm] = Form.useForm();

    // Справочники
    const [nomenclatures, setNomenclatures] = useState<any[]>([]);
    const [docTypes, setDocTypes] = useState<any[]>([]);
    const [orgOptionsSender, setOrgOptionsSender] = useState<{ value: string; label: string }[]>([]);
    const [orgOptionsRecipient, setOrgOptionsRecipient] = useState<{ value: string; label: string }[]>([]);

    const loadRefs = async () => {
        try {
            const { GetActiveForDirection } = await import('../../wailsjs/go/services/NomenclatureService');
            const noms = await GetActiveForDirection('incoming');
            setNomenclatures(noms || []);

            const { GetDocumentTypes } = await import('../../wailsjs/go/services/ReferenceService');
            const types = await GetDocumentTypes();
            setDocTypes(types || []);
        } catch (err) {
            console.error('Failed to load refs:', err);
        }
    };

    const onSenderOrgSearch = async (query: string) => {
        if (query.length < 2) { setOrgOptionsSender(query ? [{ value: query, label: query }] : []); return; }
        try {
            const { SearchOrganizations } = await import('../../wailsjs/go/services/ReferenceService');
            const orgs = await SearchOrganizations(query);
            const items = (orgs || []).map((o: any) => ({ value: o.name, label: o.name }));
            if (!items.find((i: any) => i.value === query)) items.unshift({ value: query, label: query });
            setOrgOptionsSender(items);
        } catch { setOrgOptionsSender([{ value: query, label: query }]); }
    };

    const onRecipientOrgSearch = async (query: string) => {
        if (query.length < 2) { setOrgOptionsRecipient(query ? [{ value: query, label: query }] : []); return; }
        try {
            const { SearchOrganizations } = await import('../../wailsjs/go/services/ReferenceService');
            const orgs = await SearchOrganizations(query);
            const items = (orgs || []).map((o: any) => ({ value: o.name, label: o.name }));
            if (!items.find((i: any) => i.value === query)) items.unshift({ value: query, label: query });
            setOrgOptionsRecipient(items);
        } catch { setOrgOptionsRecipient([{ value: query, label: query }]); }
    };

    const load = async () => {
        setLoading(true);
        try {
            const { GetList } = await import('../../wailsjs/go/services/IncomingDocumentService');
            const result = await GetList({
                search, page, pageSize,
                nomenclatureId: '', documentTypeId: '', orgId: '',
                dateFrom: filterDateFrom, dateTo: filterDateTo,
                incomingNumber: filterIncomingNumber,
                outgoingNumber: filterOutgoingNumber,
                senderName: filterSenderName,
                outgoingDateFrom: filterOutDateFrom,
                outgoingDateTo: filterOutDateTo,
                resolution: filterNoResolution ? '' : filterResolution,
                noResolution: filterNoResolution,
                nomenclatureIds: currentRole === 'executor'
                    ? ((user?.department?.nomenclatureIds?.length) ? user.department.nomenclatureIds : ['00000000-0000-0000-0000-000000000000'])
                    : filterNomenclatureIds,
            });
            setData(result?.items || []);
            setTotalCount(result?.totalCount || 0);
        } catch (err: any) {
            message.error(err?.message || String(err));
        }
        setLoading(false);
    };

    useEffect(() => { loadRefs(); }, []);
    useEffect(() => { load(); }, [page, search, filterIncomingNumber, filterOutgoingNumber, filterSenderName, filterDateFrom, filterDateTo, filterOutDateFrom, filterOutDateTo, filterResolution, filterNoResolution, filterNomenclatureIds]);

    const clearFilters = () => {
        setSearch(''); setFilterIncomingNumber(''); setFilterOutgoingNumber('');
        setFilterSenderName(''); setFilterDateFrom(''); setFilterDateTo('');
        setFilterOutDateFrom(''); setFilterOutDateTo('');
        setFilterResolution(''); setFilterNoResolution(false);
        setFilterNomenclatureIds([]); setPage(1);
    };
    const hasFilters = filterIncomingNumber || filterOutgoingNumber || filterSenderName || filterDateFrom || filterDateTo || filterOutDateFrom || filterOutDateTo || filterResolution || filterNoResolution || filterNomenclatureIds.length > 0;

    // Регистрация
    const onRegister = async (values: any) => {
        try {
            const { Register } = await import('../../wailsjs/go/services/IncomingDocumentService');
            await Register(
                values.nomenclatureId, values.documentTypeId,
                values.senderOrgName, values.recipientOrgName,
                values.incomingDate?.format('YYYY-MM-DD') || '',
                values.outgoingDateSender?.format('YYYY-MM-DD') || '',
                values.outgoingNumberSender || '',
                values.intermediateNumber || '',
                values.intermediateDate?.format('YYYY-MM-DD') || '',
                values.subject,
                values.content || '', values.pagesCount || 1,
                values.senderSignatory || '', values.senderExecutor || '',
                values.addressee || '', values.resolution || '',
            );
            message.success('Документ зарегистрирован');
            setRegisterModalOpen(false); registerForm.resetFields(); load();
        } catch (err: any) { message.error(err?.message || String(err)); }
    };

    // Редактирование
    const onEdit = async (values: any) => {
        try {
            const { Update } = await import('../../wailsjs/go/services/IncomingDocumentService');
            await Update(
                editDoc.id, values.documentTypeId,
                values.senderOrgName, values.recipientOrgName,
                values.outgoingDateSender?.format('YYYY-MM-DD') || '',
                values.outgoingNumberSender || '',
                values.intermediateNumber || '',
                values.intermediateDate?.format('YYYY-MM-DD') || '',
                values.subject,
                values.content || '', values.pagesCount || 1,
                values.senderSignatory || '', values.senderExecutor || '',
                values.addressee || '', values.resolution || '',
            );
            message.success('Документ обновлён');
            setEditModalOpen(false); editForm.resetFields(); setEditDoc(null); load();
        } catch (err: any) { message.error(err?.message || String(err)); }
    };

    const openEdit = (record: any) => {
        setEditDoc(record);
        editForm.setFieldsValue({
            documentTypeId: record.documentTypeId,
            senderOrgName: record.senderOrgName,
            recipientOrgName: record.recipientOrgName,
            outgoingNumberSender: record.outgoingNumberSender,
            outgoingDateSender: record.outgoingDateSender ? dayjs(record.outgoingDateSender) : null,
            subject: record.subject, content: record.content,
            pagesCount: record.pagesCount, senderSignatory: record.senderSignatory,
            senderExecutor: record.senderExecutor, addressee: record.addressee,
            intermediateNumber: record.intermediateNumber || '',
            intermediateDate: record.intermediateDate ? dayjs(record.intermediateDate) : null,
            resolution: record.resolution || '',
        });
        setEditModalOpen(true);
    };

    const onDelete = async (id: string) => {
        try {
            const { Delete } = await import('../../wailsjs/go/services/IncomingDocumentService');
            await Delete(id); message.success('Удалено'); load();
        } catch (err: any) { message.error(err?.message || String(err)); }
    };

    const columns = [
        {
            title: 'Номер / Дата',
            key: 'number',
            width: 140,
            render: (_: any, r: any) => (
                <div>
                    <div style={{ fontWeight: 600 }}>{r.incomingNumber}</div>
                    <div style={{ fontSize: 12, color: '#888' }}>
                        от {dayjs(r.incomingDate).format('DD.MM.YYYY')}
                    </div>
                </div>
            )
        },
        {
            title: 'Отправитель',
            key: 'sender',
            width: 250,
            render: (_: any, r: any) => (
                <div>
                    <div style={{ fontWeight: 600 }}>{r.senderOrgName}</div>
                    {(r.outgoingNumberSender || r.outgoingDateSender) && (
                        <div style={{ fontSize: 12, color: '#666' }}>
                            Исх: {r.outgoingNumberSender} {r.outgoingDateSender ? `от ${dayjs(r.outgoingDateSender).format('DD.MM.YYYY')}` : ''}
                        </div>
                    )}
                </div>
            )
        },
        {
            title: 'Краткое содержание',
            key: 'content',
            render: (_: any, r: any) => (
                <div>
                    <div style={{ fontWeight: 500 }}>{r.subject}</div>
                </div>
            )
        },
        {
            title: 'Адресат / Резолюция',
            key: 'addressee',
            width: 220,
            render: (_: any, r: any) => (
                <div style={{ fontSize: 12 }}>
                    <div>Кому: {r.addressee}</div>
                    {r.resolution && <div style={{ marginTop: 4, fontStyle: 'italic', color: '#555' }}>Рез: {r.resolution}</div>}
                </div>
            )
        },
        {
            title: 'Действия',
            key: 'actions',
            width: 120,
            render: (_: any, record: any) => (
                <Space>
                    <Button size="small" icon={<EyeOutlined />} onClick={() => { setViewDocId(record.id); setViewModalOpen(true); }} />
                    {!isExecutorOnly && (
                        <>
                            <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} />
                            <Popconfirm title="Удалить документ?" onConfirm={() => onDelete(record.id)}>
                                <Button size="small" icon={<DeleteOutlined />} danger />
                            </Popconfirm>
                        </>
                    )}
                </Space>
            ),
        },
    ];

    // Общая форма
    const renderDocForm = (form: any, isEdit: boolean) => (
        <Form form={form} layout="vertical" onFinish={isEdit ? onEdit : onRegister}>
            {!isEdit && (
                <Row gutter={16}>
                    <Col span={12}>
                        <Form.Item name="nomenclatureId" label="Дело (номенклатура)" rules={[{ required: true }]}>
                            <Select placeholder="Выберите дело">
                                {nomenclatures.map((n: any) => (
                                    <Select.Option key={n.id} value={n.id}>{n.index} — {n.name}</Select.Option>
                                ))}
                            </Select>
                        </Form.Item>
                    </Col>
                    <Col span={12}>
                        <Form.Item name="incomingDate" label="Дата регистрации" rules={[{ required: true }]}>
                            <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" />
                        </Form.Item>
                    </Col>
                </Row>
            )}
            <Row gutter={16}>
                <Col span={isEdit ? 24 : 12}>
                    <Form.Item name="documentTypeId" label="Тип документа" rules={[{ required: true }]}>
                        <Select placeholder="Выберите тип">
                            {docTypes.map((t: any) => (
                                <Select.Option key={t.id} value={t.id}>{t.name}</Select.Option>
                            ))}
                        </Select>
                    </Form.Item>
                </Col>
            </Row>
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
            <Form.Item name="subject" label="Краткое содержание" rules={[{ required: true }]}>
                <Input />
            </Form.Item>
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
                        <Select showSearch filterOption={false} onSearch={onSenderOrgSearch} options={orgOptionsSender} notFoundContent={null}
                            onInputKeyDown={(e) => { if (e.key === ' ') e.stopPropagation(); }} />
                    </Form.Item>
                </Col>
                <Col span={12}>
                    <Form.Item name="recipientOrgName" label="Организация-получатель" rules={[{ required: true }]}>
                        <Select showSearch filterOption={false} onSearch={onRecipientOrgSearch} options={orgOptionsRecipient} notFoundContent={null}
                            onInputKeyDown={(e) => { if (e.key === ' ') e.stopPropagation(); }} />
                    </Form.Item>
                </Col>
            </Row>
            <Row gutter={16}>
                <Col span={8}>
                    <Form.Item name="senderSignatory" label="Подписант" rules={[{ required: true, message: 'Укажите подписанта' }]}>
                        <Input />
                    </Form.Item>
                </Col>
                <Col span={8}>
                    <Form.Item name="senderExecutor" label="Исполнитель">
                        <Input />
                    </Form.Item>
                </Col>
                <Col span={8}>
                    <Form.Item name="addressee" label="Кому адресован" rules={[{ required: true, message: 'Укажите адресата' }]}>
                        <Input />
                    </Form.Item>
                </Col>
            </Row>
            <Form.Item name="resolution" label="Резолюция">
                <TextArea rows={2} placeholder="Текст резолюции" />
            </Form.Item>
            <Form.Item name="content" label="Содержание (подробно)">
                <TextArea rows={2} />
            </Form.Item>
        </Form>
    );

    return (
        <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                <Title level={4} style={{ margin: 0 }}>Входящие документы</Title>
                <Space>
                    {!filterDisabled && (
                        <Select
                            mode="multiple" size="middle" style={{ minWidth: 250 }}
                            placeholder="Все дела"
                            value={filterNomenclatureIds}
                            onChange={(vals: string[]) => { setFilterNomenclatureIds(vals); setPage(1); }}
                            allowClear
                            options={nomenclatures.map((n: any) => ({ value: n.id, label: `${n.index} — ${n.name}` }))}
                        />
                    )}
                    <Input.Search placeholder="Поиск по содержанию..." allowClear onSearch={setSearch} style={{ width: 250 }} prefix={<SearchOutlined />} />
                    {!isExecutorOnly && (
                        <Button type="primary" icon={<PlusOutlined />} onClick={() => {
                            registerForm.resetFields();
                            registerForm.setFieldsValue({ incomingDate: dayjs(), pagesCount: 1 });
                            setRegisterModalOpen(true);
                        }}>Зарегистрировать</Button>
                    )}
                </Space>
            </div>

            {/* Панель фильтров */}
            <Collapse
                size="small"
                style={{ marginBottom: 16 }}
                items={[{
                    key: 'filters',
                    label: <span><FilterOutlined /> Расширенный поиск {hasFilters ? <Tag color="blue" style={{ marginLeft: 8 }}>Активны</Tag> : null}</span>,
                    children: (
                        <div>
                            <Row gutter={16}>
                                <Col span={6}>
                                    <div style={{ marginBottom: 8 }}>
                                        <Text type="secondary" style={{ fontSize: 12 }}>Вх. номер</Text>
                                        <Input size="small" value={filterIncomingNumber} onChange={e => { setFilterIncomingNumber(e.target.value); setPage(1); }} placeholder="Рег. номер" allowClear />
                                    </div>
                                </Col>
                                <Col span={6}>
                                    <div style={{ marginBottom: 8 }}>
                                        <Text type="secondary" style={{ fontSize: 12 }}>Исх. номер</Text>
                                        <Input size="small" value={filterOutgoingNumber} onChange={e => { setFilterOutgoingNumber(e.target.value); setPage(1); }} placeholder="Исх. номер" allowClear />
                                    </div>
                                </Col>
                                <Col span={6}>
                                    <div style={{ marginBottom: 8 }}>
                                        <Text type="secondary" style={{ fontSize: 12 }}>Отправитель</Text>
                                        <Input size="small" value={filterSenderName} onChange={e => { setFilterSenderName(e.target.value); setPage(1); }} placeholder="Название организации" allowClear />
                                    </div>
                                </Col>
                                <Col span={6} style={{ display: 'flex', alignItems: 'flex-end', paddingBottom: 8 }}>
                                    {hasFilters && (
                                        <Button size="small" icon={<ClearOutlined />} onClick={clearFilters}>Сбросить фильтры</Button>
                                    )}
                                </Col>
                            </Row>
                            <Row gutter={16}>
                                <Col span={12}>
                                    <div style={{ marginBottom: 8 }}>
                                        <Text type="secondary" style={{ fontSize: 12 }}>Дата получения (диапазон)</Text>
                                        <RangePicker
                                            size="small" style={{ width: '100%' }} format="DD.MM.YYYY"
                                            value={filterDateFrom && filterDateTo ? [dayjs(filterDateFrom), dayjs(filterDateTo)] : null}
                                            onChange={(dates) => {
                                                setFilterDateFrom(dates?.[0]?.format('YYYY-MM-DD') || '');
                                                setFilterDateTo(dates?.[1]?.format('YYYY-MM-DD') || '');
                                                setPage(1);
                                            }}
                                        />
                                    </div>
                                </Col>
                                <Col span={12}>
                                    <div style={{ marginBottom: 8 }}>
                                        <Text type="secondary" style={{ fontSize: 12 }}>Дата отправки (диапазон)</Text>
                                        <RangePicker
                                            size="small" style={{ width: '100%' }} format="DD.MM.YYYY"
                                            value={filterOutDateFrom && filterOutDateTo ? [dayjs(filterOutDateFrom), dayjs(filterOutDateTo)] : null}
                                            onChange={(dates) => {
                                                setFilterOutDateFrom(dates?.[0]?.format('YYYY-MM-DD') || '');
                                                setFilterOutDateTo(dates?.[1]?.format('YYYY-MM-DD') || '');
                                                setPage(1);
                                            }}
                                        />
                                    </div>
                                </Col>
                            </Row>
                            <Row gutter={16}>
                                <Col span={8}>
                                    <div style={{ marginBottom: 8 }}>
                                        <Text type="secondary" style={{ fontSize: 12 }}>Резолюция</Text>
                                        <Input size="small" value={filterResolution} onChange={e => { setFilterResolution(e.target.value); setPage(1); }} placeholder="Текст резолюции" allowClear disabled={filterNoResolution} />
                                    </div>
                                </Col>
                                <Col span={8} style={{ display: 'flex', alignItems: 'flex-end', paddingBottom: 8 }}>
                                    <label style={{ fontSize: 12, cursor: 'pointer' }}>
                                        <input type="checkbox" checked={filterNoResolution} onChange={e => { setFilterNoResolution(e.target.checked); setPage(1); }} style={{ marginRight: 6 }} />
                                        Без резолюции
                                    </label>
                                </Col>
                            </Row>
                        </div>
                    ),
                }]}
            />

            <Table columns={columns} dataSource={data} rowKey="id" loading={loading} size="small"
                pagination={{
                    current: page, pageSize, total: totalCount,
                    onChange: (p, ps) => { setPage(p); setPageSize(ps); },
                    showSizeChanger: true, pageSizeOptions: ['10', '20', '50']
                }}
            />

            {/* Регистрация */}
            <Modal title="Регистрация входящего документа" open={registerModalOpen}
                onCancel={() => setRegisterModalOpen(false)} onOk={() => registerForm.submit()} width={700} okText="Зарегистрировать">
                {renderDocForm(registerForm, false)}
            </Modal>

            {/* Редактирование */}
            <Modal title={`Редактирование: ${editDoc?.incomingNumber || ''}`} open={editModalOpen}
                onCancel={() => { setEditModalOpen(false); setEditDoc(null); }} onOk={() => editForm.submit()} width={700} okText="Сохранить">
                {renderDocForm(editForm, true)}
            </Modal>

            {/* Просмотр */}
            <DocumentViewModal
                open={viewModalOpen}
                onCancel={() => setViewModalOpen(false)}
                documentId={viewDocId}
                documentType="incoming"
            />
        </div>
    );
};

export default IncomingPage;
