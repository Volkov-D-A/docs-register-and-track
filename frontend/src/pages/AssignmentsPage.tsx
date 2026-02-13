import React, { useState, useEffect } from 'react';
import {
    Typography, Table, Button, Modal, Form, Input, Select, DatePicker,
    Space, Row, Col, Tag, message, Popconfirm, Tabs, Tooltip
} from 'antd';
import {
    PlusOutlined, SearchOutlined, EditOutlined, DeleteOutlined,
    CheckCircleOutlined, PlayCircleOutlined, CloseCircleOutlined, UndoOutlined,
    FilterOutlined, ClearOutlined
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useAuthStore } from '../store/useAuthStore';
import AssignmentModal from '../components/AssignmentModal';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { RangePicker } = DatePicker;

const AssignmentsPage: React.FC = () => {
    const { user, hasRole, currentRole } = useAuthStore();
    const [activeTab, setActiveTab] = useState('inbox');
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [totalCount, setTotalCount] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(20);

    // Filters
    const [search, setSearch] = useState('');
    const [filterStatus, setFilterStatus] = useState('');
    const [filterDateFrom, setFilterDateFrom] = useState('');
    const [filterDateTo, setFilterDateTo] = useState('');
    const [filterExecutorId, setFilterExecutorId] = useState('');

    // Modals
    const [modalOpen, setModalOpen] = useState(false);
    const [editAssignment, setEditAssignment] = useState<any>(null);
    const [reportModalOpen, setReportModalOpen] = useState(false);
    const [currentAssignmentId, setCurrentAssignmentId] = useState<string>('');
    const [reportText, setReportText] = useState('');

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

            if (activeTab === 'inbox') {
                executorId = user?.id || '';
            } else if (activeTab === 'all') {
                executorId = filterExecutorId; // From filter
            }

            const result = await GetList({
                page, pageSize,
                search,
                status: filterStatus,
                dateFrom: filterDateFrom,
                dateTo: filterDateTo,
                executorId: executorId,
            });
            setData(result?.items || []);
            setTotalCount(result?.totalCount || 0);
        } catch (err: any) {
            message.error(err?.message || String(err));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { load(); }, [page, pageSize, activeTab, search, filterStatus, filterDateFrom, filterDateTo, filterExecutorId]);

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
        setPage(1);
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
                    <Tag style={{ marginTop: 4, fontSize: 10 }}>{r.documentType === 'incoming' ? 'Входящий' : 'Исходящий'}</Tag>
                </div>
            )
        },
        { title: 'Поручение', dataIndex: 'content', key: 'content' },
        { title: 'Исполнитель', dataIndex: 'executorName', key: 'executorName', width: 140 },
        {
            title: 'Срок', dataIndex: 'deadline', key: 'deadline', width: 90,
            render: (v: string) => v ? dayjs(v).format('DD.MM.YYYY') : '',
        },
        {
            title: 'Статус', dataIndex: 'status', key: 'status', width: 110,
            render: (status: string) => {
                let color = 'default';
                let text = status;
                switch (status) {
                    case 'new': color = 'blue'; text = 'Новое'; break;
                    case 'in_progress': color = 'orange'; text = 'В работе'; break;
                    case 'completed': color = 'green'; text = 'Исполнено'; break;
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
                const canEdit = isAdmin;

                return (
                    <Space size={2}>
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
                                <Button size="small" icon={<CheckCircleOutlined />} type="primary" ghost
                                    onClick={() => { setCurrentAssignmentId(r.id); setReportText(r.report || ''); setReportModalOpen(true); }} />
                            </Tooltip>
                        )}
                        {isAdmin && r.status === 'completed' && (
                            <Tooltip title="Вернуть на доработку">
                                <Button size="small" icon={<UndoOutlined />} onClick={() => updateStatus(r.id, 'returned')} />
                            </Tooltip>
                        )}
                        {isAdmin && r.status !== 'cancelled' && r.status !== 'completed' && (
                            <Tooltip title="Отменить">
                                <Button size="small" icon={<CloseCircleOutlined />} danger onClick={() => updateStatus(r.id, 'cancelled')} />
                            </Tooltip>
                        )}
                    </Space>
                );
            }
        },
    ];

    const tabItems = [
        { key: 'inbox', label: 'Мне на исполнение', icon: <PlayCircleOutlined /> },
    ];
    if (hasRole('admin')) {
        tabItems.push({ key: 'all', label: 'Все поручения', icon: <FilterOutlined /> });
    }

    return (
        <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                <Title level={4} style={{ margin: 0 }}>Поручения</Title>
                <Input.Search placeholder="Поиск..." allowClear onSearch={setSearch} style={{ width: 250 }} prefix={<SearchOutlined />} />
            </div>

            <Tabs activeKey={activeTab} onChange={key => { setActiveTab(key); setPage(1); }} items={tabItems.map(t => ({
                key: t.key, label: <span>{t.icon} {t.label}</span>
            }))} />

            {/* Фильтры */}
            <div style={{ marginBottom: 16, padding: '12px', background: '#f5f5f5', borderRadius: 6 }}>
                <Row gutter={16} align="middle">
                    <Col span={5}>
                        <Select style={{ width: '100%' }} placeholder="Статус" allowClear value={filterStatus} onChange={setFilterStatus}>
                            <Select.Option value="new">Новое</Select.Option>
                            <Select.Option value="in_progress">В работе</Select.Option>
                            <Select.Option value="completed">Исполнено</Select.Option>
                            <Select.Option value="returned">Возврат</Select.Option>
                            <Select.Option value="cancelled">Отменено</Select.Option>
                        </Select>
                    </Col>
                    <Col span={7}>
                        <RangePicker style={{ width: '100%' }} format="DD.MM.YYYY"
                            value={filterDateFrom && filterDateTo ? [dayjs(filterDateFrom), dayjs(filterDateTo)] : null}
                            onChange={(dates) => {
                                setFilterDateFrom(dates?.[0]?.format('YYYY-MM-DD') || '');
                                setFilterDateTo(dates?.[1]?.format('YYYY-MM-DD') || '');
                            }}
                        />
                    </Col>
                    {activeTab === 'all' && (
                        <>
                            <Col span={6}>
                                <Select style={{ width: '100%' }} placeholder="Исполнитель" allowClear showSearch
                                    options={executors.map(u => ({ value: u.id, label: u.fullName }))}
                                    value={filterExecutorId} onChange={setFilterExecutorId}
                                />
                            </Col>
                        </>
                    )}
                    <Col span={2}>
                        <Button icon={<ClearOutlined />} onClick={clearFilters} />
                    </Col>
                </Row>
            </div>

            <Table
                columns={columns} dataSource={data} rowKey="id"
                loading={loading} size="small"
                pagination={{
                    current: page, pageSize, total: totalCount,
                    onChange: (p, ps) => { setPage(p); setPageSize(ps); }
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
        </div>
    );
};

export default AssignmentsPage;
