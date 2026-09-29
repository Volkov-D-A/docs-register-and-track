import React, { useEffect, useState } from 'react';
import { Modal, Form, Input, Select, DatePicker, App } from 'antd';
import dayjs from 'dayjs';
import { models } from '../../wailsjs/go/models';
import { emitAssignmentsChanged } from '../events/assignmentEvents';
import { formatAppError } from '../utils/appError';

/**
 * Свойства модального окна создания задачи на ознакомление.
 */
interface AcknowledgmentModalProps {
    open: boolean;
    onCancel: () => void;
    onSuccess: () => void;
    documentId: string;
    initialValues?: any;
}

/**
 * Модальное окно для отправки документа на ознакомление.
 * @param open Флаг открытия модального окна
 * @param onCancel Обработчик отмены
 * @param onSuccess Обработчик успешного создания задачи
 * @param documentId Идентификатор документа
 */
const AcknowledgmentModal: React.FC<AcknowledgmentModalProps> = ({ open, onCancel, onSuccess, documentId, initialValues }) => {
    const { message } = App.useApp();
    const [form] = Form.useForm();
    const [loading, setLoading] = useState(false);
    const [users, setUsers] = useState<any[]>([]);

    useEffect(() => {
        if (open) {
            loadUsers();
            form.resetFields();
            if (initialValues) form.setFieldsValue({ content: initialValues.content, userIds: (initialValues.users || []).map((u: any) => u.userId), deadline: initialValues.deadline ? dayjs(initialValues.deadline) : null });
        }
    }, [form, open, initialValues]);

    const loadUsers = async () => {
        try {
            const { GetExecutors } = await import('../../wailsjs/go/services/UserService');
            const data = await GetExecutors();
            setUsers(data || []);
        } catch (err) {
            console.error(err);
        }
    };

    const handleOk = async () => {
        if (loading) {
            return;
        }
        try {
            const values = await form.validateFields();
            setLoading(true);

            const { CreateTask, Update } = await import('../../wailsjs/go/services/AssignmentService');
            const deadline = values.deadline?.format('YYYY-MM-DD') || '';
            if (initialValues) await Update(initialValues.id, '', values.content || '', deadline, []);
            else await CreateTask(models.AssignmentRequest.createFrom({ type: 'acknowledgment', documentId, content: values.content || '', deadline, userIds: values.userIds }));
            emitAssignmentsChanged({ documentId });

            message.success(initialValues ? 'Ознакомление изменено' : 'Задача создана');
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
            title={initialValues ? "Редактирование ознакомления" : "На ознакомление"}
            open={open}
            onCancel={onCancel}
            onOk={handleOk}
            confirmLoading={loading}
        >
            <Form form={form} layout="vertical">
                <Form.Item
                    name="userIds"
                    label="Сотрудники"
                    rules={[{ required: true, message: 'Выберите сотрудников' }]}
                >
                    <Select
                        mode="multiple"
                        disabled={!!initialValues}
                        placeholder="Выберите сотрудников"
                        optionFilterProp="children"
                        filterOption={(input, option) =>
                            (option?.children as unknown as string).toLowerCase().indexOf(input.toLowerCase()) >= 0
                        }
                    >
                        {[...users, ...(initialValues?.users || []).filter((recipient: any) => !users.some(user => user.id === recipient.userId)).map((recipient: any) => ({ id: recipient.userId, fullName: recipient.userName }))].map(u => (
                            <Select.Option key={u.id} value={u.id}>
                                {u.fullName}
                            </Select.Option>
                        ))}
                    </Select>
                </Form.Item>

                <Form.Item name="deadline" label="Срок ознакомления"><DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" /></Form.Item>
                <Form.Item name="content" label="Содержание / Комментарий">
                    <Input.TextArea rows={4} />
                </Form.Item>
            </Form>
        </Modal>
    );
};

export default AcknowledgmentModal;
