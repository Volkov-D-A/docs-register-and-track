import React, { useState, useEffect } from 'react';
import {
    Typography, Table, Button, Modal, Form, Input, Select, DatePicker,
    InputNumber, Space, Row, Col, Tag, message, Popconfirm, Collapse, Tabs,
} from 'antd';
import AssignmentList from '../components/AssignmentList';
import AcknowledgmentList from '../components/AcknowledgmentList';
import FileListComponent from '../components/FileListComponent';
import { LinksTab } from '../components/DocumentLinks/LinksTab';

import {
    PlusOutlined, SearchOutlined, EyeOutlined, DeleteOutlined, EditOutlined,
    FilterOutlined, ClearOutlined
} from '@ant-design/icons';
import dayjs from 'dayjs';
import locale from 'antd/es/date-picker/locale/ru_RU';

import { useAuthStore } from '../store/useAuthStore';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { RangePicker } = DatePicker;

const OutgoingPage: React.FC = () => {
    const { user, currentRole } = useAuthStore();
    const isExecutorOnly = currentRole === 'executor';
    // Скрываем фильтр, если пользователь — исполнитель без админских прав
    const filterDisabled = isExecutorOnly;

    // Данные
    const [data, setData] = useState<any[]>([]);
    const [totalCount, setTotalCount] = useState(0);
    const [loading, setLoading] = useState(false);

    // Пагинация
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(10);

    // Справочники
    const [nomenclatures, setNomenclatures] = useState<any[]>([]);
    const [docTypes, setDocTypes] = useState<any[]>([]);
    const [orgOptionsRecipient, setOrgOptionsRecipient] = useState<any[]>([]);

    // Фильтры
    const [search, setSearch] = useState('');
    const [filterNomenclatureIds, setFilterNomenclatureIds] = useState<string[]>([]);
    const [filterOutgoingNumber, setFilterOutgoingNumber] = useState('');
    const [filterRecipientName, setFilterRecipientName] = useState('');
    const [filterDateFrom, setFilterDateFrom] = useState('');
    const [filterDateTo, setFilterDateTo] = useState('');

    // Модалки
    const [registerModalOpen, setRegisterModalOpen] = useState(false);
    const [editModalOpen, setEditModalOpen] = useState(false);
    const [viewDoc, setViewDoc] = useState<any>(null);
    const [viewModalOpen, setViewModalOpen] = useState(false);
    const [editDoc, setEditDoc] = useState<any>(null);

    const [registerForm] = Form.useForm();
    const [editForm] = Form.useForm();

    // Загрузка справочников
    const loadRefs = async () => {
        try {
            const { GetActiveForDirection } = await import('../../wailsjs/go/services/NomenclatureService');
            const { GetDocumentTypes } = await import('../../wailsjs/go/services/ReferenceService');

            const noms = await GetActiveForDirection('outgoing'); // Исходящие
            setNomenclatures(noms || []);

            const types = await GetDocumentTypes();
            setDocTypes(types || []);
        } catch (e) {
            console.error(e);
        }
    };

    // Поиск организаций
    const onRecipientOrgSearch = async (val: string) => {
        if (!val) return;
        try {
            const { SearchOrganizations } = await import('../../wailsjs/go/services/ReferenceService');
            const res = await SearchOrganizations(val);
            const options = res ? res.map((r: any) => ({ value: r.name, label: r.name })) : [];
            // Добавляем введенное значение, если его нет
            if (val && !options.find((o: any) => o.value === val)) {
                options.unshift({ value: val, label: val });
            }
            setOrgOptionsRecipient(options);
        } catch (e) { console.error(e); }
    };

    // Загрузка списка
    const load = async () => {
        setLoading(true);
        try {
            const { GetList } = await import('../../wailsjs/go/services/OutgoingDocumentService');
            const result = await GetList({
                search, page, pageSize,
                nomenclatureIds: currentRole === 'executor'
                    ? ((user?.department?.nomenclatureIds?.length) ? user.department.nomenclatureIds : ['00000000-0000-0000-0000-000000000000'])
                    : filterNomenclatureIds,
                documentTypeId: '', orgId: '',
                dateFrom: filterDateFrom, dateTo: filterDateTo,
                outgoingNumber: filterOutgoingNumber,
                recipientName: filterRecipientName,
            });
            setData(result?.items || []);
            setTotalCount(result?.totalCount || 0);
        } catch (err: any) {
            message.error(err?.message || String(err));
        }
        setLoading(false);
    };

    useEffect(() => { loadRefs(); }, []);
    useEffect(() => { load(); }, [page, search, filterNomenclatureIds, filterOutgoingNumber, filterRecipientName, filterDateFrom, filterDateTo]);

    const clearFilters = () => {
        setSearch(''); setFilterNomenclatureIds([]);
        setFilterOutgoingNumber(''); setFilterRecipientName('');
        setFilterDateFrom(''); setFilterDateTo('');
        setPage(1);
    };

    const hasFilters = filterNomenclatureIds.length > 0 || filterOutgoingNumber || filterRecipientName || filterDateFrom || filterDateTo;

    // Регистрация
    const onRegister = async (values: any) => {
        try {
            const { Register } = await import('../../wailsjs/go/services/OutgoingDocumentService');
            await Register(
                values.nomenclatureId, values.documentTypeId,
                values.recipientOrgName, values.addressee,
                values.outgoingDate?.format('YYYY-MM-DD') || '',
                values.subject, values.content, values.pagesCount,
                values.senderSignatory, values.senderExecutor
            );
            message.success('Документ зарегистрирован');
            setRegisterModalOpen(false);
            registerForm.resetFields();
            load();
        } catch (err: any) {
            message.error(err?.message || String(err));
        }
    };

    // Редактирование
    const onUpdate = async (values: any) => {
        try {
            const { Update } = await import('../../wailsjs/go/services/OutgoingDocumentService');
            await Update(
                editDoc.id, values.documentTypeId,
                values.recipientOrgName, values.addressee,
                values.outgoingDate?.format('YYYY-MM-DD') || '',
                values.subject, values.content, values.pagesCount,
                values.senderSignatory, values.senderExecutor
            );
            message.success('Документ обновлен');
            setEditModalOpen(false);
            editForm.resetFields();
            setEditDoc(null);
            load();
        } catch (err: any) {
            message.error(err?.message || String(err));
        }
    };

    // Удаление
    const onDelete = async (id: string) => {
        try {
            const { Delete } = await import('../../wailsjs/go/services/OutgoingDocumentService');
            await Delete(id);
            message.success('Удалено');
            load();
        } catch (err: any) {
            message.error(err?.message || String(err));
        }
    };

    const columns = [
        {
            title: 'Номер / Дата',
            key: 'number',
            width: 140,
            render: (_: any, r: any) => (
                <div>
                    <div style={{ fontWeight: 600 }}>{r.outgoingNumber}</div>
                    <div style={{ fontSize: 12, color: '#888' }}>
                        от {dayjs(r.outgoingDate).format('DD.MM.YYYY')}
                    </div>

                </div>
            )
        },
        {
            title: 'Получатель',
            key: 'recipient',
            width: 250,
            render: (_: any, r: any) => (
                <div>
                    <div style={{ fontWeight: 600 }}>{r.recipientOrgName}</div>
                    <div style={{ fontSize: 12 }}>Адресат: {r.addressee}</div>
                </div>
            )
        },
        {
            title: 'Краткое содержание',
            key: 'content',
            render: (_: any, r: any) => (
                <div>
                    <div style={{ fontWeight: 500 }}>{r.subject}</div>
                    <div style={{ fontSize: 12, color: '#666' }}>{r.documentTypeName}</div>
                </div>
            )
        },
        {
            title: 'Исполнитель / Подписант',
            key: 'executor',
            width: 200,
            render: (_: any, r: any) => (
                <div style={{ fontSize: 12 }}>
                    <div>Исп: {r.senderExecutor}</div>
                    <div>Подп: {r.senderSignatory}</div>
                </div>
            )
        },
        {
            title: 'Действия',
            key: 'actions',
            width: 120,
            render: (_: any, r: any) => (
                <Space>
                    <Button size="small" icon={<EyeOutlined />} onClick={() => { setViewDoc(r); setViewModalOpen(true); }} />
                    {!isExecutorOnly && (
                        <>
                            <Button size="small" icon={<EditOutlined />} onClick={() => {
                                setEditDoc(r);
                                editForm.setFieldsValue({
                                    ...r,
                                    outgoingDate: dayjs(r.outgoingDate),
                                });
                                setEditModalOpen(true);
                            }} />
                            <Popconfirm title="Удалить?" onConfirm={() => onDelete(r.id)}>
                                <Button size="small" icon={<DeleteOutlined />} danger />
                            </Popconfirm>
                        </>
                    )}
                </Space>
            )
        }
    ];

    return (
        <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                <Title level={4} style={{ margin: 0 }}>Исходящие документы</Title>
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
                            registerForm.setFieldsValue({ outgoingDate: dayjs(), pagesCount: 1 });
                            setRegisterModalOpen(true);
                        }}>Зарегистрировать</Button>
                    )}
                </Space>
            </div>

            <Collapse
                size="small"
                style={{ marginBottom: 16 }}
                items={[{
                    key: 'filters',
                    label: <span><FilterOutlined /> Расширенный поиск {hasFilters ? <Tag color="blue" style={{ marginLeft: 8 }}>Активны</Tag> : null}</span>,
                    children: (
                        <div>
                            <Row gutter={16}>
                                <Col span={8}>
                                    <div style={{ marginBottom: 8 }}>
                                        <Text type="secondary" style={{ fontSize: 12 }}>Исх. номер</Text>
                                        <Input size="small" value={filterOutgoingNumber} onChange={e => { setFilterOutgoingNumber(e.target.value); setPage(1); }} placeholder="Исх. номер" allowClear />
                                    </div>
                                </Col>
                                <Col span={8}>
                                    <div style={{ marginBottom: 8 }}>
                                        <Text type="secondary" style={{ fontSize: 12 }}>Получатель</Text>
                                        <Input size="small" value={filterRecipientName} onChange={e => { setFilterRecipientName(e.target.value); setPage(1); }} placeholder="Организация" allowClear />
                                    </div>
                                </Col>
                                <Col span={8}>
                                    <div style={{ marginBottom: 8 }}>
                                        <Text type="secondary" style={{ fontSize: 12 }}>Дата (диапазон)</Text>
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
                            </Row>
                            {hasFilters && (
                                <div style={{ marginTop: 8, display: 'flex', justifyContent: 'flex-end' }}>
                                    <Button size="small" icon={<ClearOutlined />} onClick={clearFilters}>Сбросить фильтры</Button>
                                </div>
                            )}
                        </div>
                    )
                }]}
            />

            <Table columns={columns} dataSource={data} rowKey="id" loading={loading} size="small"
                pagination={{
                    current: page, pageSize, total: totalCount,
                    onChange: (p, ps) => { setPage(p); setPageSize(ps); },
                    showSizeChanger: true, pageSizeOptions: ['10', '20', '50']
                }}
            />

            {/* Модалка регистрации */}
            <Modal
                title="Регистрация исходящего документа"
                open={registerModalOpen}
                onCancel={() => setRegisterModalOpen(false)}
                onOk={() => registerForm.submit()}
                width={800}
                confirmLoading={loading}
            >
                <Form form={registerForm} layout="vertical" onFinish={onRegister}>
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="nomenclatureId" label="Номенклатура дел" rules={[{ required: true, message: 'Выберите дело' }]}>
                                <Select options={nomenclatures.map(n => ({ value: n.id, label: `${n.index} — ${n.name}` }))} placeholder="Выберите дело" />
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
                            <Form.Item name="documentTypeId" label="Вид документа" rules={[{ required: true }]}>
                                <Select options={docTypes.map(t => ({ value: t.id, label: t.name }))} />
                            </Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item name="pagesCount" label="Кол-во листов">
                                <InputNumber style={{ width: '100%' }} min={1} />
                            </Form.Item>
                        </Col>
                    </Row>

                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="recipientOrgName" label="Получатель (Организация)" rules={[{ required: true }]}>
                                <Select showSearch onSearch={onRecipientOrgSearch} options={orgOptionsRecipient} notFoundContent={null}
                                    onInputKeyDown={(e) => { if (e.key === ' ' && !e.isDefaultPrevented()) e.stopPropagation(); }} />
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
                            <Form.Item name="senderExecutor" label="Исполнитель" rules={[{ required: true }]}>
                                <Input />
                            </Form.Item>
                        </Col>
                    </Row>

                    <Form.Item name="subject" label="Краткое содержание" rules={[{ required: true }]}>
                        <TextArea rows={2} />
                    </Form.Item>
                    <Form.Item name="content" label="Содержание / Текст">
                        <TextArea rows={4} />
                    </Form.Item>
                </Form>
            </Modal>

            {/* Модалка редактирования */}
            <Modal
                title="Редактирование документа"
                open={editModalOpen}
                onCancel={() => { setEditModalOpen(false); setEditDoc(null); }}
                onOk={() => editForm.submit()}
                width={800}
                confirmLoading={loading}
            >
                <Form form={editForm} layout="vertical" onFinish={onUpdate}>
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="outgoingDate" label="Исходящая дата" rules={[{ required: true }]}>
                                <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
                            </Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item name="pagesCount" label="Кол-во листов">
                                <InputNumber style={{ width: '100%' }} min={1} />
                            </Form.Item>
                        </Col>
                    </Row>
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="documentTypeId" label="Вид документа" rules={[{ required: true }]}>
                                <Select options={docTypes.map(t => ({ value: t.id, label: t.name }))} />
                            </Form.Item>
                        </Col>
                    </Row>
                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item name="recipientOrgName" label="Получатель (Организация)" rules={[{ required: true }]}>
                                <Select showSearch onSearch={onRecipientOrgSearch} options={orgOptionsRecipient} notFoundContent={null}
                                    onInputKeyDown={(e) => { if (e.key === ' ' && !e.isDefaultPrevented()) e.stopPropagation(); }} />
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
                            <Form.Item name="senderExecutor" label="Исполнитель" rules={[{ required: true }]}>
                                <Input />
                            </Form.Item>
                        </Col>
                    </Row>
                    <Form.Item name="subject" label="Краткое содержание" rules={[{ required: true }]}>
                        <TextArea rows={2} />
                    </Form.Item>
                    <Form.Item name="content" label="Содержание / Текст">
                        <TextArea rows={4} />
                    </Form.Item>
                </Form>
            </Modal>

            {/* Просмотр */}
            <Modal
                title={`Исходящий документ №${viewDoc?.outgoingNumber}`}
                open={viewModalOpen}
                onCancel={() => setViewModalOpen(false)}
                footer={[<Button key="close" onClick={() => setViewModalOpen(false)}>Закрыть</Button>]}
                width={700}
            >
                {viewDoc && (
                    <Tabs defaultActiveKey="info" items={[
                        {
                            key: 'info', label: 'Информация',
                            children: (
                                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                                    <Row gutter={16}>
                                        <Col span={12}>
                                            <Text type="secondary" style={{ fontSize: 12 }}>Дата:</Text> <Text strong>{dayjs(viewDoc.outgoingDate).format('DD.MM.YYYY')}</Text>
                                        </Col>
                                        <Col span={12}>
                                            <Text type="secondary" style={{ fontSize: 12 }}>Вид:</Text> <Tag>{viewDoc.documentTypeName}</Tag>
                                        </Col>
                                    </Row>
                                    <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Номенклатура:</Text> {viewDoc.nomenclatureName}</Col></Row>

                                    <div style={{ height: 1, background: '#f0f0f0', margin: '4px 0' }} />

                                    <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Получатель:</Text> {viewDoc.recipientOrgName}</Col></Row>
                                    <Row><Col span={24}><Text type="secondary" style={{ fontSize: 12 }}>Адресат:</Text> {viewDoc.addressee}</Col></Row>

                                    <div style={{ height: 1, background: '#f0f0f0', margin: '4px 0' }} />

                                    <Row gutter={16}>
                                        <Col span={12}>
                                            <Text type="secondary" style={{ fontSize: 12 }}>Подписал:</Text> {viewDoc.senderSignatory}
                                        </Col>
                                        <Col span={12}>
                                            <Text type="secondary" style={{ fontSize: 12 }}>Исполнитель:</Text> {viewDoc.senderExecutor}
                                        </Col>
                                    </Row>

                                    <div style={{ height: 1, background: '#f0f0f0', margin: '4px 0' }} />

                                    <div>
                                        <Text type="secondary" style={{ fontSize: 12 }}>Краткое содержание:</Text>
                                        <div style={{ fontWeight: 500, lineHeight: 1.2 }}>{viewDoc.subject}</div>
                                    </div>

                                    {viewDoc.content && (
                                        <div>
                                            <Text type="secondary" style={{ fontSize: 12 }}>Содержание:</Text>
                                            <div style={{ whiteSpace: 'pre-wrap', fontSize: 13, maxHeight: 100, overflowY: 'auto', background: '#fafafa', padding: 8, borderRadius: 4 }}>{viewDoc.content}</div>
                                        </div>
                                    )}
                                </div>
                            )
                        },
                        {
                            key: 'assignments', label: 'Поручения',
                            children: <AssignmentList documentId={viewDoc.id} documentType="outgoing" />
                        },
                        {
                            key: 'files', label: 'Файлы',
                            children: <FileListComponent documentId={viewDoc.id} documentType="outgoing" readOnly={false} />
                        },
                        {
                            key: 'links', label: 'Связи',
                            children: <LinksTab documentId={viewDoc.id} documentType="outgoing" documentNumber={viewDoc.outgoingNumber} />
                        },
                        {
                            key: 'acknowledgments', label: 'Ознакомление',
                            children: <AcknowledgmentList documentId={viewDoc.id} documentType="outgoing" />
                        }

                    ]} />
                )}
            </Modal>
        </div>
    );
};

export default OutgoingPage;
