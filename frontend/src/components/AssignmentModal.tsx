import React, { useEffect, useState } from 'react';
import { Modal, Form, Input, Select, DatePicker, App } from 'antd';
import dayjs from 'dayjs';

interface AssignmentModalProps {
    open: boolean;
    onCancel: () => void;
    onSuccess: () => void;
    documentId: string;
    documentType: 'incoming' | 'outgoing';
    initialValues?: any; // If editing
    isEdit: boolean;
}

const { TextArea } = Input;

const AssignmentModal: React.FC<AssignmentModalProps> = ({
    open, onCancel, onSuccess, documentId, documentType, initialValues, isEdit
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
            }
        }
    }, [open, isEdit, initialValues, form]);

    const handleSubmit = async (values: any) => {
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
            } else {
                const { Create } = await import('../../wailsjs/go/services/AssignmentService');
                await Create(
                    documentId,
                    documentType,
                    values.executorId,
                    values.content,
                    values.deadline?.format('YYYY-MM-DD') || '',
                    values.coExecutorIds || []
                );
                message.success('Поручение создано');
            }
            onSuccess();
            onCancel();
        } catch (err: any) {
            message.error(err?.message || String(err));
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

                <Form.Item name="deadline" label="Срок исполнения">
                    <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" />
                </Form.Item>

                <Form.Item name="content" label="Текст поручения" rules={[{ required: true, message: 'Введите текст' }]}>
                    <TextArea rows={3} placeholder="Что нужно сделать..." />
                </Form.Item>
            </Form>
        </Modal>
    );
};

export default AssignmentModal;
