import React, { useEffect, useState } from 'react';
import { Modal, Form, Input, Select, DatePicker, App, Switch, InputNumber, Alert } from 'antd';
import dayjs from 'dayjs';
import { formatAppError } from '../utils/appError';
import { emitAssignmentsChanged } from '../events/assignmentEvents';

/**
 * Свойства модального окна создания/редактирования поручения.
 */
interface AssignmentModalProps {
    open: boolean;
    onCancel: () => void;
    onSuccess: () => void;
    documentId: string;
    initialValues?: any; // If editing
    isEdit: boolean;
}

const { TextArea } = Input;

/**
 * Модальное окно для создания и редактирования поручений.
 * @param open Флаг открытия модального окна
 * @param onCancel Обработчик отмены
 * @param onSuccess Обработчик успешного создания/редактирования
 * @param documentId Идентификатор документа
 * @param initialValues Начальные значения (при редактировании)
 * @param isEdit Флаг режима редактирования
 */
const AssignmentModal: React.FC<AssignmentModalProps> = ({
    open, onCancel, onSuccess, documentId, initialValues, isEdit
}) => {
    const { message } = App.useApp();
    const [form] = Form.useForm();
    const [executors, setExecutors] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);

    const loadExecutors = async () => {
        try {
            const { GetExecutors } = await import('../../wailsjs/go/services/UserService');
            const users = await GetExecutors();
            setExecutors(users || []);
        } catch (err) {
            console.error('Failed to load executors', err);
        }
    };

    useEffect(() => {
        if (open) {
            loadExecutors();
            form.resetFields();
            if (isEdit && initialValues) {
                form.setFieldsValue({
                    executorId: initialValues.executorId,
                    controllerId: initialValues.controllerId,
                    content: initialValues.content,
                    deadline: initialValues.deadline ? dayjs(initialValues.deadline) : null,
                    coExecutorIds: initialValues.coExecutorIds || (initialValues.coExecutors?.map((u: any) => u.id) || []),
                });
            } else {
                form.setFieldsValue({ recurring: false, schedulePreset: 'monthly_first', intervalUnit: 'month', intervalValue: 1, dayRule: 'fixed', dayOfMonth: 1 });
            }
        }
    }, [open, isEdit, initialValues, form]);

    const handleSubmit = async (values: any) => {
        if (loading) {
            return;
        }
        setLoading(true);
        try {
            if (isEdit) {
                const { Update } = await import('../../wailsjs/go/services/AssignmentService');
                await Update(
                    initialValues.id,
                    values.executorId,
                    values.content,
                    values.deadline?.format('YYYY-MM-DD') || '',
                    values.coExecutorIds || []
                );
                message.success('Поручение обновлено');
                emitAssignmentsChanged({ documentId: initialValues.documentId || documentId });
            } else {
                const assignmentService = await import('../../wailsjs/go/services/AssignmentService');
                if (values.recurring) {
                    const schedule = resolveSchedule(values);
                    await assignmentService.CreateSeries({
                        documentId,
                        executorId: values.executorId,
                        content: values.content,
                        firstDeadline: values.deadline?.format('YYYY-MM-DD') || '',
                        intervalUnit: schedule.intervalUnit,
                        intervalValue: schedule.intervalValue,
                        dayRule: schedule.dayRule,
                        dayOfMonth: schedule.dayOfMonth,
                        coExecutorIds: values.coExecutorIds || [],
                    } as any);
                } else {
                    await assignmentService.Create(
                        documentId,
                        values.executorId,
                        values.content,
                        values.deadline?.format('YYYY-MM-DD') || '',
                        values.coExecutorIds || []
                    );
                }
                message.success('Поручение создано');
                emitAssignmentsChanged({ documentId });
            }
            onSuccess();
            onCancel();
        } catch (err: unknown) {
            message.error(formatAppError(err));
        } finally {
            setLoading(false);
        }
    };

    return (
        <Modal
            title={isEdit ? "Редактирование поручения" : "Новое поручение"}
            open={open}
            onCancel={onCancel}
            onOk={() => form.submit()}
            confirmLoading={loading}
            okText={isEdit ? "Сохранить" : "Создать"}
        >
            <Form form={form} layout="vertical" onFinish={handleSubmit}>
                {!isEdit && (
                    <Form.Item name="recurring" label="Регулярное поручение" valuePropName="checked">
                        <Switch checkedChildren="Да" unCheckedChildren="Нет" />
                    </Form.Item>
                )}
                <Form.Item name="executorId" label="Ответственный исполнитель" rules={[{ required: true, message: 'Выберите исполнителя' }]}>
                    <Select placeholder="Выберите сотрудника" showSearch filterOption={(input, option) =>
                        (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                    }
                        options={executors.map((u: any) => ({ value: u.id, label: u.fullName }))}
                    />
                </Form.Item>

                <Form.Item shouldUpdate={(prev, curr) => prev.executorId !== curr.executorId}>
                    {({ getFieldValue }) => {
                        const executorId = getFieldValue('executorId');
                        return (
                            <Form.Item name="coExecutorIds" label="Соисполнители">
                                <Select
                                    mode="multiple"
                                    placeholder="Выберите соисполнителей"
                                    showSearch
                                    filterOption={(input, option) =>
                                        (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                                    }
                                    options={executors
                                        .filter((u: any) => u.id !== executorId) // Exclude main executor
                                        .map((u: any) => ({ value: u.id, label: u.fullName }))}
                                />
                            </Form.Item>
                        );
                    }}
                </Form.Item>

                <Form.Item shouldUpdate={(prev, curr) => prev.recurring !== curr.recurring || prev.schedulePreset !== curr.schedulePreset || prev.intervalUnit !== curr.intervalUnit || prev.dayRule !== curr.dayRule} noStyle>
                    {({ getFieldValue }) => {
                        const recurring = !isEdit && getFieldValue('recurring');
                        const preset = getFieldValue('schedulePreset');
                        return (
                            <>
                                {recurring && (
                                    <>
                                        <Form.Item name="schedulePreset" label="Расписание" rules={[{ required: true }]}>
                                            <Select options={[
                                                { value: 'monthly_first', label: 'Первое число каждого месяца' },
                                                { value: 'monthly_15', label: '15-е число каждого месяца' },
                                                { value: 'monthly_last', label: 'Последний день каждого месяца' },
                                                { value: 'quarter_end', label: 'Последний день каждого квартала' },
                                                { value: 'custom', label: 'Своё расписание' },
                                            ]} />
                                        </Form.Item>
                                        {preset === 'custom' && (
                                            <div style={{ display: 'flex', gap: 12 }}>
                                                <Form.Item name="intervalValue" label="Каждые N" rules={[{ required: true }]} style={{ flex: 1 }}>
                                                    <InputNumber min={1} max={3650} style={{ width: '100%' }} />
                                                </Form.Item>
                                                <Form.Item name="intervalUnit" label="Период" rules={[{ required: true }]} style={{ flex: 1 }}>
                                                    <Select options={[{ value: 'day', label: 'Дней' }, { value: 'week', label: 'Недель' }, { value: 'month', label: 'Месяцев' }, { value: 'year', label: 'Лет' }]} />
                                                </Form.Item>
                                                {['month', 'year'].includes(getFieldValue('intervalUnit')) && <Form.Item name="dayRule" label="Правило дня" rules={[{ required: true }]} style={{ flex: 1 }}>
                                                    <Select options={[{ value: 'fixed', label: 'Число месяца' }, { value: 'last_day', label: 'Последний день' }]} />
                                                </Form.Item>}
                                                {['month', 'year'].includes(getFieldValue('intervalUnit')) && getFieldValue('dayRule') === 'fixed' && (
                                                    <Form.Item name="dayOfMonth" label="Число" rules={[{ required: true }]} style={{ width: 80 }}>
                                                        <InputNumber min={1} max={31} style={{ width: '100%' }} />
                                                    </Form.Item>
                                                )}
                                            </div>
                                        )}
                                        <Alert type="info" showIcon message="После принятия текущей итерации следующая появится сразу с очередным календарным сроком." style={{ marginBottom: 16 }} />
                                    </>
                                )}
                                <Form.Item
                                    name="deadline"
                                    label={recurring ? 'Первый плановый срок' : 'Срок исполнения'}
                                    rules={recurring ? [{ required: true, message: 'Укажите первый плановый срок' }, { validator: (_, value) => validateDeadlineForPreset(value, preset) }] : []}
                                >
                                    <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" />
                                </Form.Item>
                            </>
                        );
                    }}
                </Form.Item>

                <Form.Item name="content" label="Текст поручения" rules={[{ required: true, message: 'Введите текст' }]}>
                    <TextArea rows={3} placeholder="Что нужно сделать..." />
                </Form.Item>
            </Form>
        </Modal>
    );
};

const resolveSchedule = (values: any) => {
    switch (values.schedulePreset) {
        case 'monthly_first': return { intervalUnit: 'month', intervalValue: 1, dayRule: 'fixed', dayOfMonth: 1 };
        case 'monthly_15': return { intervalUnit: 'month', intervalValue: 1, dayRule: 'fixed', dayOfMonth: 15 };
        case 'monthly_last': return { intervalUnit: 'month', intervalValue: 1, dayRule: 'last_day', dayOfMonth: 0 };
        case 'quarter_end': return { intervalUnit: 'month', intervalValue: 3, dayRule: 'last_day', dayOfMonth: 0 };
        default: return { intervalUnit: values.intervalUnit, intervalValue: values.intervalValue, dayRule: ['day', 'week'].includes(values.intervalUnit) ? 'same_day' : values.dayRule, dayOfMonth: values.dayRule === 'fixed' && !['day', 'week'].includes(values.intervalUnit) ? values.dayOfMonth : 0 };
    }
};

const validateDeadlineForPreset = (_value: unknown, preset: string) => {
    const value = _value as dayjs.Dayjs | null;
    if (!value) return Promise.resolve();
    if (preset === 'monthly_first' && value.date() !== 1) return Promise.reject(new Error('Выберите первое число месяца'));
    if (preset === 'monthly_15' && value.date() !== 15) return Promise.reject(new Error('Выберите 15-е число месяца'));
    if ((preset === 'monthly_last' || preset === 'quarter_end') && value.date() !== value.daysInMonth()) return Promise.reject(new Error('Выберите последний день месяца'));
    if (preset === 'quarter_end' && ![2, 5, 8, 11].includes(value.month())) return Promise.reject(new Error('Выберите конец марта, июня, сентября или декабря'));
    return Promise.resolve();
};

export default AssignmentModal;
