import React, { useEffect, useState } from 'react';
import { Alert, App, Button, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tabs, Tag } from 'antd';
import dayjs from 'dayjs';
import { formatAppError } from '../utils/appError';
import { emitAssignmentsChanged } from '../events/assignmentEvents';

const { TextArea } = Input;

type Props = {
    open: boolean;
    seriesId: string;
    documentId: string;
    onCancel: () => void;
    onSuccess: () => void | Promise<void>;
};

const statusLabels: Record<string, { text: string; color: string }> = {
    new: { text: 'Новое', color: 'blue' },
    in_progress: { text: 'В работе', color: 'orange' },
    completed: { text: 'Исполнено', color: 'green' },
    finished: { text: 'Завершён', color: 'geekblue' },
    returned: { text: 'Возврат', color: 'volcano' },
    cancelled: { text: 'Отменено', color: 'red' },
};

const AssignmentSeriesModal: React.FC<Props> = ({ open, seriesId, documentId, onCancel, onSuccess }) => {
    const { message } = App.useApp();
    const [form] = Form.useForm();
    const [series, setSeries] = useState<any>(null);
    const [history, setHistory] = useState<any[]>([]);
    const [executors, setExecutors] = useState<any[]>([]);
    const [filesByAssignment, setFilesByAssignment] = useState<Record<string, any[]>>({});
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (!open || !seriesId) return;
        let active = true;
        setLoading(true);
        Promise.all([
            import('../../wailsjs/go/services/AssignmentService').then((service) => Promise.all([service.GetSeries(seriesId), service.GetSeriesHistory(seriesId)])),
            import('../../wailsjs/go/services/UserService').then((service) => service.GetExecutors()),
        ]).then(([[loadedSeries, loadedHistory], users]) => {
            if (!active) return;
            setSeries(loadedSeries);
            setHistory(loadedHistory || []);
            setExecutors(users || []);
            form.setFieldsValue({
                executorId: loadedSeries.executorId,
                coExecutorIds: loadedSeries.coExecutorIds || [],
                content: loadedSeries.content,
                intervalUnit: loadedSeries.intervalUnit,
                intervalValue: loadedSeries.intervalValue,
                dayRule: loadedSeries.dayRule,
                dayOfMonth: loadedSeries.dayOfMonth || 1,
            });
        }).catch((error) => message.error(formatAppError(error))).finally(() => active && setLoading(false));
        return () => { active = false; };
    }, [form, message, open, seriesId]);

    const save = async () => {
        const values = await form.validateFields();
        setLoading(true);
        try {
            const service = await import('../../wailsjs/go/services/AssignmentService');
            const updated = await service.UpdateSeries(seriesId, {
                documentId,
                executorId: values.executorId,
                content: values.content,
                firstDeadline: '',
                intervalUnit: values.intervalUnit,
                intervalValue: values.intervalValue,
                dayRule: ['day', 'week'].includes(values.intervalUnit) ? 'same_day' : values.dayRule,
                dayOfMonth: values.dayRule === 'fixed' && !['day', 'week'].includes(values.intervalUnit) ? values.dayOfMonth : 0,
                coExecutorIds: values.coExecutorIds || [],
            } as any);
            setSeries(updated);
            message.success('Параметры будущих итераций обновлены');
            emitAssignmentsChanged({ documentId });
            await onSuccess();
        } catch (error) {
            message.error(formatAppError(error));
        } finally {
            setLoading(false);
        }
    };

    const cancelSeries = async () => {
        setLoading(true);
        try {
            const service = await import('../../wailsjs/go/services/AssignmentService');
            await service.CancelSeries(seriesId);
            message.success('Серия отменена. Текущая итерация сохранена.');
            emitAssignmentsChanged({ documentId });
            await onSuccess();
            onCancel();
        } catch (error) {
            message.error(formatAppError(error));
        } finally {
            setLoading(false);
        }
    };

    const loadFiles = async (assignmentId: string) => {
        if (filesByAssignment[assignmentId]) return;
        try {
            const service = await import('../../wailsjs/go/services/AttachmentService');
            const files = await service.GetAssignmentFiles(assignmentId);
            setFilesByAssignment((current) => ({ ...current, [assignmentId]: files || [] }));
        } catch (error) {
            message.error(formatAppError(error));
        }
    };

    const parameters = (
        <Form form={form} layout="vertical">
            {!series?.active && <Alert type="warning" showIcon message="Серия отменена. Новые итерации создаваться не будут." style={{ marginBottom: 16 }} />}
            <Form.Item name="executorId" label="Ответственный исполнитель" rules={[{ required: true }]}>
                <Select showSearch optionFilterProp="label" options={executors.map((user) => ({ value: user.id, label: user.fullName }))} disabled={!series?.active} />
            </Form.Item>
            <Form.Item shouldUpdate={(previous, current) => previous.executorId !== current.executorId} noStyle>
                {({ getFieldValue }) => <Form.Item name="coExecutorIds" label="Соисполнители">
                    <Select mode="multiple" showSearch optionFilterProp="label" options={executors.filter((user) => user.id !== getFieldValue('executorId')).map((user) => ({ value: user.id, label: user.fullName }))} disabled={!series?.active} />
                </Form.Item>}
            </Form.Item>
            <Form.Item name="content" label="Текст будущих итераций" rules={[{ required: true }]}><TextArea rows={3} disabled={!series?.active} /></Form.Item>
            <Space align="start" style={{ display: 'flex' }}>
                <Form.Item name="intervalValue" label="Каждые N" rules={[{ required: true }]}><InputNumber min={1} max={3650} disabled={!series?.active} /></Form.Item>
                <Form.Item name="intervalUnit" label="Период" rules={[{ required: true }]}><Select style={{ width: 140 }} options={[{ value: 'day', label: 'Дней' }, { value: 'week', label: 'Недель' }, { value: 'month', label: 'Месяцев' }, { value: 'year', label: 'Лет' }]} disabled={!series?.active} /></Form.Item>
                <Form.Item shouldUpdate={(previous, current) => previous.intervalUnit !== current.intervalUnit || previous.dayRule !== current.dayRule} noStyle>
                    {({ getFieldValue }) => ['month', 'year'].includes(getFieldValue('intervalUnit')) ? <>
                        <Form.Item name="dayRule" label="Плановый день" rules={[{ required: true }]}><Select style={{ width: 180 }} options={[{ value: 'fixed', label: 'Число месяца' }, { value: 'last_day', label: 'Последний день месяца' }]} disabled={!series?.active} /></Form.Item>
                        {getFieldValue('dayRule') === 'fixed' ? <Form.Item name="dayOfMonth" label="Число" rules={[{ required: true }]}><InputNumber min={1} max={31} disabled={!series?.active} /></Form.Item> : null}
                    </> : null}
                </Form.Item>
            </Space>
            <Alert type="info" showIcon message="Изменения применятся только к итерациям, которые ещё не созданы." />
        </Form>
    );

    const historyTab = <Table size="small" rowKey="id" dataSource={history} pagination={false} columns={[
        { title: '№', dataIndex: 'iterationNumber', width: 54 },
        { title: 'Срок', dataIndex: 'plannedDeadline', width: 110, render: (value: string) => value ? dayjs(value).format('DD.MM.YYYY') : '' },
        { title: 'Исполнитель', dataIndex: 'executorName' },
        { title: 'Статус', dataIndex: 'status', width: 120, render: (value: string) => { const status = statusLabels[value] || { text: value, color: 'default' }; return <Tag color={status.color}>{status.text}</Tag>; } },
        { title: 'Отчёт / причина возврата', dataIndex: 'report', ellipsis: true },
    ]} expandable={{
        onExpand: (expanded, record) => { if (expanded) void loadFiles(record.id); },
        expandedRowRender: (record) => <div>
            <div><b>Полный отчёт:</b> {record.report || '—'}</div>
            <div style={{ marginTop: 8 }}><b>Файлы исполнения:</b>{' '}
                {filesByAssignment[record.id] === undefined ? 'загрузка…' : filesByAssignment[record.id].length === 0 ? 'нет' : filesByAssignment[record.id].map((file) => <Tag key={file.id}>{file.filename}</Tag>)}
            </div>
        </div>,
    }} />;

    return <Modal title="Управление серией поручений" open={open} onCancel={onCancel} width={900} footer={series?.active ? [
        <Popconfirm key="cancel-series" title="Отменить серию?" description="Текущая итерация сохранится, новые создаваться не будут." okText="Отменить серию" cancelText="Назад" onConfirm={cancelSeries}><Button danger>Отменить серию</Button></Popconfirm>,
        <Button key="close" onClick={onCancel}>Закрыть</Button>,
        <Button key="save" type="primary" loading={loading} onClick={save}>Сохранить</Button>,
    ] : [<Button key="close" onClick={onCancel}>Закрыть</Button>]}>
        <Tabs items={[{ key: 'parameters', label: 'Параметры', children: parameters }, { key: 'history', label: `История (${history.length})`, children: historyTab }]} />
    </Modal>;
};

export default AssignmentSeriesModal;
