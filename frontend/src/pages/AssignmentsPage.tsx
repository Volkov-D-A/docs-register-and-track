import { onServerEvent } from '../events/serverEvents';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
    Typography, Table, Button, Input, Select, DatePicker,
    Space, Tag, Popconfirm, Tooltip, Radio, App
} from 'antd';
import {
    SearchOutlined, EditOutlined, DeleteOutlined,
    CalendarOutlined, ClearOutlined, EyeOutlined, SyncOutlined, CheckOutlined
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useAuthStore } from '../store/useAuthStore';
import { useAssignmentModeStore } from '../store/useAssignmentModeStore';
import { DOCUMENT_KIND_INCOMING_LETTER } from '../constants/documentKinds';
import AcknowledgmentModal from '../components/AcknowledgmentModal';
import AssignmentModal from '../components/AssignmentModal';

import DocumentViewModal from '../components/DocumentViewModal';
import { useDocumentKindAccess } from '../hooks/useDocumentKindAccess';
import { formatAppError } from '../utils/appError';
import { onAssignmentsChanged } from '../events/assignmentEvents';
import { dto, models } from '../../wailsjs/go/models';
import { CoalescedRequest } from '../utils/coalescedRequest';
import AssignmentSeriesModal from '../components/AssignmentSeriesModal';
import { assignmentFiltersFromMetric, type AssignmentNavigation, type AssignmentMode } from '../components/assignmentNavigation';
import { GetOverview } from '../../wailsjs/go/services/WorkspaceService';

const { Title } = Typography;
const { RangePicker } = DatePicker;
const statusFilters = [
    { value: 'new', label: 'Новые', tone: 'new' },
    { value: 'in_progress', label: 'В работе', tone: 'in_progress' },
    { value: 'returned', label: 'На доработке', tone: 'returned' },
    { value: 'completed', label: 'На приёмке', tone: 'acceptance' },
] as const;

const recipientPluralRules = new Intl.PluralRules('ru');
const formatRecipientCount = (count: number) => {
    const form = recipientPluralRules.select(count);
    return `${count} ${form === 'one' ? 'адресат' : form === 'few' ? 'адресата' : 'адресатов'}`;
};

/**
 * Страница управления поручениями.
 * Позволяет просматривать, фильтровать и администрировать поручения в зависимости от роли.
 */
const AssignmentsPage: React.FC<{ initialView: AssignmentNavigation | null }> = ({ initialView }) => {
    const { message } = App.useApp();
    const { user } = useAuthStore();
    const { hasAction, hasAnyAction, ready: accessReady } = useDocumentKindAccess();
    const [data, setData] = useState<dto.Assignment[]>([]);
    const [loading, setLoading] = useState(false);
    const [totalCount, setTotalCount] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(10);
    const selectedMode = useAssignmentModeStore((state) => state.mode);
    const setMode = useAssignmentModeStore((state) => state.setMode);
    const [defaultMode, setDefaultMode] = useState<AssignmentMode>(initialView?.mode || 'execution');
    const mode = selectedMode || defaultMode;
    const [availableModes, setAvailableModes] = useState<AssignmentMode[]>([]);
    const [initialFilters] = useState(() => assignmentFiltersFromMetric(initialView?.metric));

    // Фильтры
    const [search, setSearch] = useState('');
    const [searchInput, setSearchInput] = useState('');
    const [filterTypes, setFilterTypes] = useState<string[]>(initialView?.type && initialView.type !== 'all' ? [initialView.type] : []);
    const [filterStatuses, setFilterStatuses] = useState<string[]>(initialFilters.status ? [initialFilters.status] : []);
    const [filterDateFrom, setFilterDateFrom] = useState(initialFilters.dateFrom);
    const [filterDateTo, setFilterDateTo] = useState(initialFilters.dateTo);
    const [filterExecutorId, setFilterExecutorId] = useState('');
    const [filterOverdue, setFilterOverdue] = useState(initialFilters.overdueOnly);
    const [showFinished, setShowFinished] = useState(false);
    const [periodOpen, setPeriodOpen] = useState(false);
    const periodButtonRef = useRef<HTMLButtonElement>(null);

    // Модальные окна
    const [acknowledgmentOpen, setAcknowledgmentOpen] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const [editAssignment, setEditAssignment] = useState<dto.Assignment | null>(null);
    const [seriesAssignment, setSeriesAssignment] = useState<dto.Assignment | null>(null);

    // View Document
    const [viewDocId, setViewDocId] = useState<string>('');
    const [viewDocKind, setViewDocKind] = useState(DOCUMENT_KIND_INCOMING_LETTER);
    const [viewModalOpen, setViewModalOpen] = useState(false);

    // Refs
    const [executors, setExecutors] = useState<dto.User[]>([]);
    const assignmentListRequestRef = useRef(new CoalescedRequest<{ items: dto.Assignment[]; totalCount: number }>());

    const loadUsers = async () => {
        try {
            const { GetExecutors } = await import('../../wailsjs/go/services/UserService');
            const users = await GetExecutors();
            setExecutors(users || []);
        } catch (e) { console.error(e); }
    };

    const load = useCallback(async () => {
        if (!accessReady || availableModes.length === 0) {
            return;
        }
        setLoading(true);
        return assignmentListRequestRef.current.refresh(async () => {
            const { GetList } = await import('../../wailsjs/go/services/AssignmentService');

            const executorId = mode === 'control' ? filterExecutorId : '';

            const result = await GetList(models.AssignmentFilter.createFrom({
                page,
                pageSize,
                types: filterTypes,
                search,
                statuses: filterStatuses,
                dateFrom: filterDateFrom,
                dateTo: filterDateTo,
                executorId: executorId,
                showFinished: showFinished,
                overdueOnly: filterOverdue,
                mode,
            }));
            return { items: result?.items || [], totalCount: result?.totalCount || 0 };
        }, {
            onSuccess: ({ items, totalCount }) => { setData(items); setTotalCount(totalCount); },
            onError: (err) => message.error(formatAppError(err)),
            onSettled: () => setLoading(false),
        });
    }, [
        accessReady,
        availableModes,
        filterDateFrom,
        filterDateTo,
        filterExecutorId,
        filterOverdue,
        filterStatuses,
        filterTypes,
        mode,
        message,
        page,
        pageSize,
        search,
        showFinished,
    ]);

    useEffect(() => () => assignmentListRequestRef.current.invalidate(), []);

    useEffect(() => {
        if (accessReady) {
            load();
        }
    }, [accessReady, load]);

    useEffect(() => onServerEvent((event) => {
        if (event.topic === 'resync' || event.topic === 'access-changed' || event.topic === 'documents'
            || (event.topic === 'document-changed' && (event.resource === 'assignments' || event.resource === 'document'))) {
            void load();
        }
    }), [load]);

    useEffect(() => onAssignmentsChanged(() => {
        void load();
    }), [load]);

    useEffect(() => {
        loadUsers();
    }, []);

    useEffect(() => {
        if (!accessReady || !user?.id) return;
        let active = true;
        void GetOverview('').then((result) => {
            if (!active) return;
            const modes = result.assignmentModes as AssignmentMode[];
            setAvailableModes(modes);
            const savedMode = useAssignmentModeStore.getState().mode;
            const fallback = initialView && modes.includes(initialView.mode)
                ? initialView.mode : modes[0];
            if (fallback) {
                setDefaultMode(fallback);
                if (savedMode && !modes.includes(savedMode)) setMode(fallback);
            } else if (savedMode) {
                setMode('');
            }
        }).catch((error) => {
            if (active) message.error(formatAppError(error, 'Не удалось определить режимы поручений'));
        });
        return () => { active = false; };
    }, [accessReady, user?.id, initialView, message, setMode]);

    const onDelete = async (id: string) => {
        try {
            const { Delete } = await import('../../wailsjs/go/services/AssignmentService');
            await Delete(id);
            message.success('Поручение удалено');
            load();
        } catch (err: unknown) {
            message.error(formatAppError(err));
        }
    };

    const clearFilters = () => {
        setSearch(''); setSearchInput(''); setFilterStatuses([]); setFilterTypes([]);
        setFilterDateFrom(''); setFilterDateTo('');
        setFilterExecutorId('');
        setFilterOverdue(false);
        setShowFinished(false);
        setPeriodOpen(false);
        setPage(1);
    };

    const setDateFilterToday = () => {
        setPeriodOpen(false);
        setPage(1);
        const today = dayjs().format('YYYY-MM-DD');
        setFilterDateFrom(today);
        setFilterDateTo(today);
        setFilterOverdue(false);
    };

    const setDateFilterLess3Days = () => {
        setPeriodOpen(false);
        setPage(1);
        const today = dayjs().format('YYYY-MM-DD');
        const next3Days = dayjs().add(3, 'day').format('YYYY-MM-DD');
        setFilterDateFrom(today);
        setFilterDateTo(next3Days);
        setFilterOverdue(false);
    };

    const setDateFilterOverdue = () => {
        setPeriodOpen(false);
        setPage(1);
        setFilterDateFrom('');
        setFilterDateTo('');
        setFilterOverdue(true);
    };

    const handleViewDocument = (id: string, kind: string) => {
        setViewDocId(id);
        setViewDocKind(kind);
        setViewModalOpen(true);
    };

    const columns = [
        {
            title: 'Дата', dataIndex: 'createdAt', key: 'createdAt', width: 96, className: 'assignments-date-column',
            render: (v: string) => dayjs(v).format('DD.MM.YYYY'),
        },
        {
            title: 'Документ', key: 'doc', width: '25%',
            render: (_: unknown, r: dto.Assignment) => (
                <div className="assignments-document-cell" title={r.documentSubject || r.documentNumber || 'Без номера'}>
                    <div style={{ fontWeight: 600 }}>{r.documentNumber || 'Без номера'}</div>
                    <div className="assignments-document-subject">{r.documentSubject}</div>
                </div>
            )
        },
        {
            title: 'Поручение',
            dataIndex: 'content',
            key: 'content',
            width: '25%',
            ellipsis: true,
            render: (value: string, r: dto.Assignment) => (
                <div className="assignments-content-cell" title={value}>
                    {r.type === 'acknowledgment' && <Tag>Ознакомление</Tag>}{value}
                </div>
            )
        },
        {
            title: 'Исполнитель / адресаты', key: 'executorName', width: 148,
            render: (_: unknown, r: dto.Assignment) => (
                <div>
                    <div>{r.type === 'acknowledgment' ? (r.users?.length === 1 ? r.users[0].userName : formatRecipientCount((r.users || []).length)) : r.executorName}</div>
                    {r.coExecutors && r.coExecutors.length > 0 && (
                        <div style={{ fontSize: '11px', color: 'var(--app-text-muted)' }}>
                            + {r.coExecutors.map((u) => u.fullName).join(', ')}
                        </div>
                    )}
                </div>
            )
        },
        {
            title: 'Срок', dataIndex: 'deadline', key: 'deadline', width: 104,
            render: (v: string) => v ? dayjs(v).format('DD.MM.YYYY') : '',
        },
        {
            title: 'Статус', dataIndex: 'status', key: 'status', width: 110,
            render: (status: string, record: dto.Assignment) => {
                if (record.type === 'acknowledgment') { const users = record.users || []; return <Tag color={record.status === 'finished' ? 'green' : 'orange'}>{record.status === 'finished' ? 'Ознакомлены' : 'Ожидает'} ({users.filter(u => u.confirmedAt).length}/{users.length})</Tag>; }
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
                    case 'finished': color = 'geekblue'; text = 'Завершён'; break;
                    case 'cancelled': color = 'red'; text = 'Отменено'; break;
                    case 'returned': color = 'volcano'; text = 'Возврат'; break;
                }
                return <Tag color={color}>{text}</Tag>;
            }
        },
        {
            title: 'Действия', key: 'actions', width: 140,
            render: (_: unknown, r: dto.Assignment) => {
                const acknowledgment = r.type === 'acknowledgment';
                const canManageAssignment = mode === 'control' && hasAction(r.documentKind, 'assign');
                const canDelete = acknowledgment ? canManageAssignment && user?.id === r.creatorId : canManageAssignment && r.status !== 'finished' && !r.seriesId;
                const canEdit = canManageAssignment && r.status !== 'finished';

                return (
                    <Space size={2}>
                        <Tooltip title="Просмотреть карточку документа">
                            <Button size="small" icon={<EyeOutlined />} onClick={() => handleViewDocument(r.documentId, r.documentKind)} />
                        </Tooltip>
                        {canManageAssignment && (
                            <>
                                {canEdit && <Button size="small" title="Редактировать поручение" icon={<EditOutlined />} onClick={() => { setEditAssignment(r); if (acknowledgment) setAcknowledgmentOpen(true); else setModalOpen(true); }} />}
                                {(r as any).seriesId && <Button size="small" title="Управление серией" icon={<SyncOutlined />} onClick={() => setSeriesAssignment(r)} />}
                                {canDelete && <Popconfirm
                                    title="Удалить поручение?"
                                    description="Это действие нельзя отменить. Поручение исчезнет из документа и списка исполнителя."
                                    okText="Удалить"
                                    cancelText="Отмена"
                                    okButtonProps={{ danger: true }}
                                    onConfirm={() => onDelete(r.id)}
                                >
                                    <Button size="small" title="Удалить поручение" icon={<DeleteOutlined />} danger />
                                </Popconfirm>}
                            </>
                        )}
                    </Space>
                );
            }
        },
    ];



    const hasFilters = showFinished || !!searchInput || filterTypes.length > 0 || !!search || filterStatuses.length > 0 || !!filterDateFrom || !!filterDateTo || !!filterExecutorId || filterOverdue;

    const today = dayjs().format('YYYY-MM-DD');
    const deadlinePreset = filterOverdue ? 'overdue'
        : !filterDateFrom && !filterDateTo ? 'all'
        : filterDateFrom === today && filterDateTo === today ? 'today'
        : filterDateFrom === today && filterDateTo === dayjs().add(3, 'day').format('YYYY-MM-DD') ? 'soon'
        : 'period';

    return (
        <div>
            <div className="assignments-page-heading">
                <Space wrap>
                    <Title level={4} style={{ margin: 0 }}>{mode === 'control' ? 'Поручения под контролем' : 'Мои поручения'}</Title>
                    {availableModes.length > 1 && <Radio.Group size="small" value={mode} options={[
                        { label: 'Исполнение', value: 'execution' },
                        { label: 'Контроль', value: 'control' },
                    ]} onChange={(event) => { setMode(event.target.value as AssignmentMode); setPage(1); }} />}
                </Space>
            </div>



            <section className="assignment-filters assignment-filters-toggles" aria-label="Статус и тип поручений">
                <div className="assignment-filter-buttons" role="group" aria-label="Статус поручения">
                    {statusFilters.map((option) => (
                        <Button key={option.value} className={`assignment-filter-button assignment-filter-button--${option.tone}`}
                            aria-pressed={filterStatuses.includes(option.value)}
                            icon={filterStatuses.includes(option.value) ? <CheckOutlined aria-hidden /> : undefined}
                            onClick={() => { setFilterStatuses((selected) => selected.includes(option.value) ? selected.filter((value) => value !== option.value) : [...selected, option.value]); setPage(1); }}>{option.label}</Button>
                    ))}
                </div>
                <Button className="assignment-filter-button assignment-filter-button--neutral"
                    aria-pressed={showFinished} icon={showFinished ? <CheckOutlined aria-hidden /> : undefined}
                    onClick={() => {
                        setShowFinished(!showFinished);
                        setPage(1);
                    }}>Показать завершённые</Button>
                <span className="assignment-filter-divider" aria-hidden="true" />
                <div className="assignment-filter-buttons" role="group" aria-label="Тип поручения">
                    {([{ value: 'execution', label: 'Исполнение', tone: 'new' }, { value: 'acknowledgment', label: 'Ознакомление', tone: 'acceptance' }] as const).map((option) => (
                        <Button key={option.value} className={`assignment-filter-button assignment-filter-button--${option.tone}`}
                            aria-pressed={filterTypes.includes(option.value)}
                            icon={filterTypes.includes(option.value) ? <CheckOutlined aria-hidden /> : undefined}
                            onClick={() => { setFilterTypes((selected) => selected.includes(option.value) ? selected.filter((value) => value !== option.value) : [...selected, option.value]); setPage(1); }}>{option.label}</Button>
                    ))}
                </div>
            </section>

            <section className="assignment-filters" aria-label="Поиск и срок поручений">
                <div className="assignment-filters-toolbar">
                    <Input.Search className="assignment-filters-search" aria-label="Поиск поручений"
                        placeholder="Поиск по поручению или документу" allowClear prefix={<SearchOutlined aria-hidden />}
                        value={searchInput} onChange={(event) => setSearchInput(event.target.value)}
                        onSearch={(value) => { setSearch(value.trim()); setPage(1); }} />
                    {mode === 'control' && hasAnyAction('assign') && (
                        <Select className="assignment-filters-executor" aria-label="Исполнитель поручения"
                            placeholder="Исполнитель / адресат" allowClear showSearch optionFilterProp="label"
                            options={executors.map((u) => ({ value: u.id, label: u.fullName }))}
                            value={filterExecutorId || undefined}
                            onChange={(value) => { setFilterExecutorId(value || ''); setPage(1); }} />
                    )}
                    <Button type="text" aria-label="Сбросить фильтры" icon={<ClearOutlined aria-hidden />}
                        disabled={!hasFilters} onClick={clearFilters}>Сбросить</Button>
                </div>
                <div className="assignment-filter-row" role="group" aria-label="Срок поручения">
                    <span className="assignment-filter-label">Срок</span>
                    <Button className="assignment-deadline-link" type="link" aria-pressed={deadlinePreset === 'all'}
                        onClick={() => { setFilterDateFrom(''); setFilterDateTo(''); setFilterOverdue(false); setPeriodOpen(false); setPage(1); }}>Любой</Button>
                    <Button className="assignment-deadline-link" type="link" aria-pressed={deadlinePreset === 'overdue'}
                        onClick={setDateFilterOverdue}>Просроченные</Button>
                    <Button className="assignment-deadline-link" type="link" aria-pressed={deadlinePreset === 'today'}
                        onClick={setDateFilterToday}>Сегодня</Button>
                    <Button className="assignment-deadline-link" type="link" aria-pressed={deadlinePreset === 'soon'}
                        onClick={setDateFilterLess3Days}>До 3 дней</Button>
                    <Button ref={periodButtonRef} className="assignment-period-button" icon={<CalendarOutlined aria-hidden />}
                        aria-pressed={deadlinePreset === 'period'} aria-expanded={periodOpen}
                        type={deadlinePreset === 'period' ? 'primary' : 'default'}
                        onClick={() => setPeriodOpen(!periodOpen)}>
                        {deadlinePreset === 'period'
                            ? `${dayjs(filterDateFrom).format('DD.MM.YYYY')} — ${dayjs(filterDateTo).format('DD.MM.YYYY')}`
                            : 'Выбрать период'}
                    </Button>
                    {periodOpen && <RangePicker className="assignment-period-picker" format="DD.MM.YYYY" autoFocus open
                        placeholder={['Срок от', 'Срок до']}
                        value={filterDateFrom && filterDateTo ? [dayjs(filterDateFrom), dayjs(filterDateTo)] : null}
                        onOpenChange={(visible) => { if (!visible) { setPeriodOpen(false); periodButtonRef.current?.focus(); } }}
                        onChange={(dates) => {
                            setFilterDateFrom(dates?.[0]?.format('YYYY-MM-DD') || '');
                            setFilterDateTo(dates?.[1]?.format('YYYY-MM-DD') || '');
                            setFilterOverdue(false);
                            setPeriodOpen(false);
                            setPage(1);
                            periodButtonRef.current?.focus();
                        }} />}
                </div>
            </section>

            <Table
                className="assignments-table"
                columns={columns} dataSource={data} rowKey="id"
                loading={loading || !accessReady || availableModes.length === 0} size="small" tableLayout="fixed"
                rowClassName={(record: dto.Assignment) => {
                    const isOverdue = record.deadline && dayjs(record.deadline).isBefore(dayjs(), 'day') && !['completed', 'finished', 'cancelled'].includes(record.status);
                    return isOverdue ? 'assignment-overdue' : '';
                }}
                pagination={{
                    current: page, pageSize, total: totalCount,
                    onChange: (p, ps) => { setPage(p); setPageSize(ps); },
                    showSizeChanger: true, pageSizeOptions: ['10', '20', '50']
                }}
                expandable={{
                    columnWidth: 28,
                    expandedRowRender: (record: dto.Assignment) => (
                        <div style={{ margin: 0 }}>
                            {record.type === 'acknowledgment' && (record.users || []).map(u => <p key={u.userId}>{u.userName}: {u.confirmedAt ? `ознакомлен ${dayjs(u.confirmedAt).format('DD.MM.YYYY HH:mm')}` : 'ожидает ознакомления'}</p>)}
                            {record.report && <p><b>{record.status === 'returned' ? 'Причина возврата:' : 'Отчет:'}</b> {record.report}</p>}
                        </div>
                    ),
                    rowExpandable: (record: dto.Assignment) => !!record.report || record.type === 'acknowledgment'
                }}
            />

            <AcknowledgmentModal open={acknowledgmentOpen} initialValues={editAssignment} documentId={editAssignment?.documentId || ''}
                onCancel={() => { setAcknowledgmentOpen(false); setEditAssignment(null); }} onSuccess={load} />
            <AssignmentModal
                open={modalOpen}
                onCancel={() => { setModalOpen(false); setEditAssignment(null); }}
                onSuccess={load}
                documentId={editAssignment?.documentId || ''}
                isEdit={true}
                initialValues={editAssignment}
            />

            <DocumentViewModal
                open={viewModalOpen}
                onCancel={() => setViewModalOpen(false)}
                documentId={viewDocId}
                documentKind={viewDocKind}
            />

            <AssignmentSeriesModal
                open={!!seriesAssignment}
                seriesId={(seriesAssignment as any)?.seriesId || ''}
                documentId={seriesAssignment?.documentId || ''}
                onCancel={() => setSeriesAssignment(null)}
                onSuccess={load}
            />
        </div>
    );



};

export default AssignmentsPage;
