import React, { useEffect, useState } from 'react';
import { Modal, Form, Input, Select, App } from 'antd';

/**
 * Свойства модального окна создания задачи на ознакомление.
 */
interface AcknowledgmentModalProps {
    open: boolean;
    onCancel: () => void;
    onSuccess: () => void;
    documentId: string;
}

/**
 * Модальное окно для отправки документа на ознакомление.
 * @param open Флаг открытия модального окна
 * @param onCancel Обработчик отмены
 * @param onSuccess Обработчик успешного создания задачи
 * @param documentId Идентификатор документа
 */
const AcknowledgmentModal: React.FC<AcknowledgmentModalProps> = ({ open, onCancel, onSuccess, documentId }) => {
    const { message } = App.useApp();
    const [form] = Form.useForm();
    const [loading, setLoading] = useState(false);
    const [users, setUsers] = useState<any[]>([]);

    useEffect(() => {
        if (open) {
            loadUsers();
            form.resetFields();
        }
    }, [open]);

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
        try {
            const values = await form.validateFields();
            setLoading(true);

            // @ts-ignore
            const { Create } = await import('../../wailsjs/go/services/AcknowledgmentService');
            await Create(documentId, values.content || '', values.userIds);

            message.success('Задача создана');
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
            title="На ознакомление"
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
                        placeholder="Выберите сотрудников"
                        optionFilterProp="children"
                        filterOption={(input, option) =>
                            (option?.children as unknown as string).toLowerCase().indexOf(input.toLowerCase()) >= 0
                        }
                    >
                        {users.map(u => (
                            <Select.Option key={u.id} value={u.id}>
                                {u.fullName}
                            </Select.Option>
                        ))}
                    </Select>
                </Form.Item>

                <Form.Item name="content" label="Содержание / Комментарий">
                    <Input.TextArea rows={4} />
                </Form.Item>
            </Form>
        </Modal>
    );
};

export default AcknowledgmentModal;
