import React, { useState, useEffect } from 'react';
import {
    Typography, Table, Button, Modal, Form, Input, Select, DatePicker,
    Space, Row, Col, Tag, message, Popconfirm, Tooltip, Switch
} from 'antd';
import {
    PlusOutlined, SearchOutlined, EditOutlined, DeleteOutlined,
    CheckCircleOutlined, PlayCircleOutlined, CloseCircleOutlined, UndoOutlined,
    ClearOutlined, EyeOutlined, FileDoneOutlined
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useAuthStore } from '../store/useAuthStore';
import AssignmentModal from '../components/AssignmentModal';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { RangePicker } = DatePicker;

const AssignmentsPage: React.FC = () => {
    const { user, currentRole, hasRole } = useAuthStore();
    // const [activeTab, setActiveTab] = useState('inbox'); // Removed
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [totalCount, setTotalCount] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(10);

    // Filters
    const [search, setSearch] = useState('');
    const [filterStatus, setFilterStatus] = useState('');
    const [filterDateFrom, setFilterDateFrom] = useState('');
    const [filterDateTo, setFilterDateTo] = useState('');
    const [filterExecutorId, setFilterExecutorId] = useState('');
    const [filterOverdue, setFilterOverdue] = useState(false);
    const [showFinished, setShowFinished] = useState(false);

    // Modals
    const [modalOpen, setModalOpen] = useState(false);
    const [editAssignment, setEditAssignment] = useState<any>(null);
    const [reportModalOpen, setReportModalOpen] = useState(false);
    const [currentAssignmentId, setCurrentAssignmentId] = useState<string>('');
    const [reportText, setReportText] = useState('');

    // View Document
    const [viewDoc, setViewDoc] = useState<any>(null);
    const [viewDocType, setViewDocType] = useState<'incoming' | 'outgoing'>('incoming');
    const [viewModalOpen, setViewModalOpen] = useState(false);

    // Refs
    const [executors, setExecutors] = useState<any[]>([]);

    const loadUsers = async () => {
        try {
            const { GetExecutors } = await import('../../wailsjs/go/services/UserService');
            const users = await GetExecutors();
            setExecutors(users || []);
        } catch (e) { console.error(e); }
    };

    const load = async () => {
        setLoading(true);
        try {
            const { GetList } = await import('../../wailsjs/go/services/AssignmentService');

            let executorId = '';
            // If currentRole is executor, force filter by user ID (own assignments only)
            // Ideally, the backend should enforce this, but frontend filtering is also good for UI state.
            // Requirement: clerk sees all, executor sees assigned.
            const canViewAll = currentRole === 'admin' || currentRole === 'clerk';

            if (canViewAll) {
                executorId = filterExecutorId;
            } else {
                executorId = user?.id || '';
            }

            const result = await GetList({
                page, pageSize,
                search,
                status: filterStatus,
                dateFrom: filterDateFrom,
                dateTo: filterDateTo,
                executorId: executorId,
                showFinished: showFinished,
                overdueOnly: filterOverdue,
            } as any); // Cast to any until models are regenerated
            setData(result?.items || []);
            setTotalCount(result?.totalCount || 0);
        } catch (err: any) {
            message.error(err?.message || String(err));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        load();
        loadUsers();
    }, [page, pageSize, currentRole, search, filterStatus, filterDateFrom, filterDateTo, filterExecutorId, showFinished, filterOverdue]);

    const updateStatus = async (id: string, status: string, report: string = '') => {
        try {
            const { UpdateStatus } = await import('../../wailsjs/go/services/AssignmentService');
            await UpdateStatus(id, status, report);
            message.success('Статус обновлен');
            load();
        } catch (err: any) {
            message.error(err?.message || String(err));
        }
    };

    const handleComplete = () => {
        if (!reportText.trim()) {
            message.error('Введите отчет об исполнении');
            return;
        }
        updateStatus(currentAssignmentId, 'completed', reportText);
        setReportModalOpen(false);
        setReportText('');
    };

    const onDelete = async (id: string) => {
        try {
            const { Delete } = await import('../../wailsjs/go/services/AssignmentService');
            await Delete(id);
            message.success('Удалено');
            load();
        } catch (err: any) {
            message.error(err?.message || String(err));
        }
    };

    const clearFilters = () => {
        setSearch(''); setFilterStatus('');
        setFilterDateFrom(''); setFilterDateTo('');
        setFilterExecutorId('');
        setFilterOverdue(false);
        setPage(1);
    };

    const setDateFilterToday = () => {
        const today = dayjs().format('YYYY-MM-DD');
        setFilterDateFrom(today);
        setFilterDateTo(today);
        setFilterOverdue(false);
    };

    const setDateFilterLess3Days = () => {
        const today = dayjs().format('YYYY-MM-DD');
        const next3Days = dayjs().add(3, 'day').format('YYYY-MM-DD');
        setFilterDateFrom(today);
        setFilterDateTo(next3Days);
        setFilterOverdue(false);
    };

    const setDateFilterOverdue = () => {
        setFilterDateFrom('');
        setFilterDateTo('');
        setFilterOverdue(true);
    };

    const handleViewDocument = async (id: string, type: string) => {
        setLoading(true);
        try {
            let doc;
            if (type === 'incoming') {
                const { GetByID } = await import('../../wailsjs/go/services/IncomingDocumentService');
                doc = await GetByID(id);
                setViewDocType('incoming');
            } else {
                const { GetByID } = await import('../../wailsjs/go/services/OutgoingDocumentService');
                doc = await GetByID(id);
                setViewDocType('outgoing');
            }
            setViewDoc(doc);
            setViewModalOpen(true);
        } catch (err: any) {
            message.error(err?.message || String(err));
        } finally {
            setLoading(false);
        }
    };

    const columns = [
        {
            title: 'Дата', dataIndex: 'createdAt', key: 'createdAt', width: 90,
            render: (v: string) => dayjs(v).format('DD.MM.YYYY'),
        },
        {
            title: 'Документ', key: 'doc', width: 220,
            render: (_: any, r: any) => (
                <div>
                    <div style={{ fontWeight: 600 }}>{r.documentNumber || 'Без номера'}</div>
                    <div style={{ fontSize: 12, color: '#666', lineHeight: '1.2' }}>{r.documentSubject}</div>
                </div>
            )
        },
        { title: 'Поручение', dataIndex: 'content', key: 'content' },
        {
            title: 'Исполнитель', key: 'executorName', width: 200,
            render: (_: any, r: any) => (
                <div>
                    <div>{r.executorName}</div>
                    {r.coExecutors && r.coExecutors.length > 0 && (
                        <div style={{ fontSize: '11px', color: '#888' }}>
                            + {r.coExecutors.map((u: any) => u.fullName).join(', ')}
                        </div>
                    )}
                </div>
            )
        },
        {
            title: 'Срок', dataIndex: 'deadline', key: 'deadline', width: 90,
            render: (v: string) => v ? dayjs(v).format('DD.MM.YYYY') : '',
        },
        {
            title: 'Статус', dataIndex: 'status', key: 'status', width: 110,
            render: (status: string, record: any) => {
                let color = 'default';
                let text = status;

                // Check for overdue completion
                const isOverdue = status === 'completed' && record.completedAt && record.deadline &&
                    dayjs(record.completedAt).isAfter(dayjs(record.deadline), 'day');

                switch (status) {
                    case 'new': color = 'blue'; text = 'Новое'; break;
                    case 'in_progress': color = 'orange'; text = 'В работе'; break;
                    case 'completed':
                        if (isOverdue) {
                            color = 'red';
                            text = 'Исполнено (просрочено)';
                        } else {
                            color = 'green';
                            text = 'Исполнено';
                        }
                        break;
                    case 'finished': color = 'geekblue'; text = 'Завершен'; break;
                    case 'cancelled': color = 'red'; text = 'Отменено'; break;
                    case 'returned': color = 'volcano'; text = 'Возврат'; break;
                }
                return <Tag color={color}>{text}</Tag>;
            }
        },
        {
            title: 'Действия', key: 'actions', width: 140,
            render: (_: any, r: any) => {
                const isExecutor = user?.id === r.executorId && currentRole === 'executor';
                const isAdmin = hasRole('admin');
                const isClerk = hasRole('clerk');

                // Admin can edit all. Clerk can edit if not finished.
                const canEdit = isAdmin || (isClerk && r.status !== 'finished');

                return (
                    <Space size={2}>
                        <Tooltip title="Просмотреть карточку документа">
                            <Button size="small" icon={<EyeOutlined />} onClick={() => handleViewDocument(r.documentId, r.documentType)} />
                        </Tooltip>
                        {canEdit && (
                            <>
                                <Button size="small" icon={<EditOutlined />} onClick={() => { setEditAssignment(r); setModalOpen(true); }} />
                                <Popconfirm title="Удалить?" onConfirm={() => onDelete(r.id)}>
                                    <Button size="small" icon={<DeleteOutlined />} danger />
                                </Popconfirm>
                            </>
                        )}
                        {isExecutor && (r.status === 'new' || r.status === 'returned') && (
                            <Tooltip title="Взять в работу">
                                <Button size="small" icon={<PlayCircleOutlined />} onClick={() => updateStatus(r.id, 'in_progress')} />
                            </Tooltip>
                        )}
                        {isExecutor && r.status === 'in_progress' && (
                            <Tooltip title="Исполнить">
                                <Button size="small" icon={<CheckCircleOutlined />}
                                    onClick={() => { setCurrentAssignmentId(r.id); setReportText(r.report || ''); setReportModalOpen(true); }} />
                            </Tooltip>
                        )}
                        {(isAdmin || hasRole('clerk')) && r.status === 'completed' && (
                            <>
                                <Tooltip title="Завершить">
                                    <Button size="small" icon={<FileDoneOutlined />} onClick={() => updateStatus(r.id, 'finished')} />
                                </Tooltip>
                                <Tooltip title="Вернуть на доработку">
                                    <Button size="small" icon={<UndoOutlined />} onClick={() => updateStatus(r.id, 'returned')} />
                                </Tooltip>
                            </>
                        )}
                        {isAdmin && r.status !== 'cancelled' && r.status !== 'finished' && r.status !== 'completed' && (
                            <Tooltip title="Отменить">
                                <Button size="small" icon={<CloseCircleOutlined />} danger onClick={() => updateStatus(r.id, 'cancelled')} />
                            </Tooltip>
                        )}
                    </Space>
                );
            }
        },
    ];



    const hasFilters = !!search || !!filterStatus || !!filterDateFrom || !!filterDateTo || !!filterExecutorId || filterOverdue;

    return (
        <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                <Title level={4} style={{ margin: 0 }}>Поручения</Title>
                <Input.Search placeholder="Поиск..." allowClear onSearch={setSearch} style={{ width: 250 }} prefix={<SearchOutlined />} />
            </div>



            {/* Фильтры */}
            <div style={{ marginBottom: 16, padding: '12px', background: '#f5f5f5', borderRadius: 6 }}>
                <Row gutter={16} align="middle">
                    <Col span={4}>
                        <Select style={{ width: '100%' }} placeholder="Статус" allowClear value={filterStatus || undefined} onChange={setFilterStatus}>
                            <Select.Option value="new">Новое</Select.Option>
                            <Select.Option value="in_progress">В работе</Select.Option>
                            <Select.Option value="completed">Исполнено</Select.Option>
                            {showFinished && <Select.Option value="finished">Завершен</Select.Option>}
                            <Select.Option value="returned">Возврат</Select.Option>
                        </Select>
                    </Col>
                    <Col span={10}>
                        <div style={{ display: 'flex', alignItems: 'center' }}>
                            <Space size="small" style={{ marginRight: 8, flexShrink: 0 }}>
                                <Button size="small" type={filterOverdue ? 'primary' : 'link'} onClick={setDateFilterOverdue} style={filterOverdue ? {} : { padding: 0 }}>Проср.</Button>
                                <Button size="small" type="link" onClick={setDateFilterToday} style={{ padding: 0 }}>Сегодня</Button>
                                <Button size="small" type="link" onClick={setDateFilterLess3Days} style={{ padding: 0 }}>&lt; 3 дней</Button>
                            </Space>
                            <RangePicker style={{ flex: 1 }} format="DD.MM.YYYY"
                                value={filterDateFrom && filterDateTo ? [dayjs(filterDateFrom), dayjs(filterDateTo)] : null}
                                onChange={(dates) => {
                                    setFilterDateFrom(dates?.[0]?.format('YYYY-MM-DD') || '');
                                    setFilterDateTo(dates?.[1]?.format('YYYY-MM-DD') || '');
                                    setFilterOverdue(false);
                                }}
                            />
                        </div>
                    </Col>
                    {(currentRole === 'admin' || currentRole === 'clerk') && (
                        <>
                            <Col span={4}>
                                <Select style={{ width: '100%' }} placeholder="Исполнитель" allowClear showSearch
                                    options={executors.map(u => ({ value: u.id, label: u.fullName }))}
                                    value={filterExecutorId || undefined} onChange={setFilterExecutorId}
                                />
                            </Col>
                        </>
                    )}
                    <Col span={4}>
                        <Space>
                            <Switch checked={showFinished} onChange={(checked) => {
                                setShowFinished(checked);
                                if (!checked && filterStatus === 'finished') {
                                    setFilterStatus('');
                                }
                            }} />
                            <Text style={{ fontSize: 12 }}>Показать<br />завершенные</Text>
                        </Space>
                    </Col>
                    <Col span={2} style={{ textAlign: 'right' }}>
                        {hasFilters && (
                            <Button icon={<ClearOutlined />} onClick={clearFilters} />
                        )}
                    </Col>
                </Row>
            </div>

            <Table
                columns={columns} dataSource={data} rowKey="id"
                loading={loading} size="small"
                rowClassName={(record) => {
                    const isOverdue = record.deadline && dayjs(record.deadline).isBefore(dayjs(), 'day') && !['completed', 'finished', 'cancelled'].includes(record.status);
                    return isOverdue ? 'assignment-overdue' : '';
                }}
                pagination={{
                    current: page, pageSize, total: totalCount,
                    onChange: (p, ps) => { setPage(p); setPageSize(ps); },
                    showSizeChanger: true, pageSizeOptions: ['10', '20', '50']
                }}
                expandable={{
                    expandedRowRender: (record) => (
                        <div style={{ margin: 0 }}>
                            {record.report && <p><b>Отчет:</b> {record.report}</p>}
                            {record.controllerName && <p><b>Контролер:</b> {record.controllerName}</p>}
                        </div>
                    ),
                    rowExpandable: (record) => !!record.report || !!record.controllerName
                }}
            />

            <AssignmentModal
                open={modalOpen}
                onCancel={() => { setModalOpen(false); setEditAssignment(null); }}
                onSuccess={load}
                documentId={editAssignment?.documentId || ''}
                documentType={editAssignment?.documentType || 'incoming'}
                isEdit={true}
                initialValues={editAssignment}
            />

            <Modal
                title="Отчет об исполнении"
                open={reportModalOpen}
                onCancel={() => setReportModalOpen(false)}
                onOk={handleComplete}
                okText="Исполнено"
            >
                <TextArea
                    rows={4}
                    value={reportText}
                    onChange={e => setReportText(e.target.value)}
                    placeholder="Введите результат выполнения поручения..."
                />
            </Modal>

            <Modal
                title={`Карточка документа (${viewDocType === 'incoming' ? 'Входящий' : 'Исходящий'})`}
                open={viewModalOpen}
                onCancel={() => setViewModalOpen(false)}
                footer={[
                    <Button key="close" type="primary" onClick={() => setViewModalOpen(false)}>Закрыть</Button>,
                ]}
                width={700}
            >
                {viewDoc && (
                    <Row gutter={[16, 12]}>
                        {viewDocType === 'incoming' ? (
                            <>
                                <Col span={12}><Text type="secondary">Рег. номер:</Text> <Text strong>{viewDoc.incomingNumber}</Text></Col>
                                <Col span={12}><Text type="secondary">Дата:</Text> <Text strong>{dayjs(viewDoc.incomingDate).format('DD.MM.YYYY')}</Text></Col>
                                <Col span={12}><Text type="secondary">Исх. №:</Text> {viewDoc.outgoingNumberSender || '—'}</Col>
                                <Col span={12}><Text type="secondary">Дата исх.:</Text> {viewDoc.outgoingDateSender ? dayjs(viewDoc.outgoingDateSender).format('DD.MM.YYYY') : '—'}</Col>
                                <Col span={24}><Text type="secondary">Тип:</Text> <Tag>{viewDoc.documentTypeName}</Tag></Col>
                                <Col span={24}><Text type="secondary">Содержание:</Text> <Text strong>{viewDoc.subject}</Text></Col>
                                <Col span={12}><Text type="secondary">Отправитель:</Text> {viewDoc.senderOrgName}</Col>
                                <Col span={12}><Text type="secondary">Получатель:</Text> {viewDoc.recipientOrgName}</Col>
                                <Col span={12}><Text type="secondary">Подписант:</Text> {viewDoc.senderSignatory || '—'}</Col>
                                <Col span={12}><Text type="secondary">Кому:</Text> {viewDoc.addressee || '—'}</Col>
                                <Col span={24}><Text type="secondary">Резолюция:</Text> <Text strong>{viewDoc.resolution || '—'}</Text></Col>
                                {viewDoc.content && <Col span={24}><Text type="secondary">Подробно:</Text><br />{viewDoc.content}</Col>}
                                <Col span={12}><Text type="secondary">Листов:</Text> {viewDoc.pagesCount}</Col>
                                <Col span={12}><Text type="secondary">Номенклатура:</Text> {viewDoc.nomenclatureName}</Col>
                            </>
                        ) : (
                            <>
                                <Col span={12}><Text type="secondary">Номер:</Text> <Text strong>{viewDoc.outgoingNumber}</Text></Col>
                                <Col span={12}><Text type="secondary">Дата:</Text> <Text strong>{dayjs(viewDoc.outgoingDate).format('DD.MM.YYYY')}</Text></Col>
                                <Col span={24}><Text type="secondary">Тип:</Text> <Tag>{viewDoc.documentTypeName}</Tag></Col>
                                <Col span={24}><Text type="secondary">Краткое содержание:</Text> <Text strong>{viewDoc.subject}</Text></Col>
                                <Col span={12}><Text type="secondary">Получатель:</Text> {viewDoc.recipientOrgName}</Col>
                                <Col span={12}><Text type="secondary">Адресат:</Text> {viewDoc.addressee}</Col>
                                <Col span={12}><Text type="secondary">Подписал:</Text> {viewDoc.senderSignatory}</Col>
                                <Col span={12}><Text type="secondary">Исполнитель:</Text> {viewDoc.senderExecutor}</Col>
                                {viewDoc.content && <Col span={24}><Text type="secondary">Содержание:</Text><br /><div style={{ whiteSpace: 'pre-wrap' }}>{viewDoc.content}</div></Col>}
                                <Col span={12}><Text type="secondary">Листов:</Text> {viewDoc.pagesCount}</Col>
                                <Col span={12}><Text type="secondary">Номенклатура:</Text> {viewDoc.nomenclatureName}</Col>
                            </>
                        )}
                    </Row>
                )}
            </Modal>
        </div>
    );
};

export default AssignmentsPage;
