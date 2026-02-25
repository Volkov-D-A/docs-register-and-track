import React, { useEffect, useState } from 'react';
import { Modal, Form, Input, Select, App } from 'antd';
import { useAuthStore } from '../store/useAuthStore';

/**
 * Свойства модального окна создания задачи на ознакомление.
 */
interface AcknowledgmentModalProps {
    open: boolean;
    onCancel: () => void;
    onSuccess: () => void;
    documentId: string;
    documentType: string;
}

/**
 * Модальное окно для отправки документа на ознакомление.
 * @param open Флаг открытия модального окна
 * @param onCancel Обработчик отмены
 * @param onSuccess Обработчик успешного создания задачи
 * @param documentId Идентификатор документа
 * @param documentType Тип документа
 */
const AcknowledgmentModal: React.FC<AcknowledgmentModalProps> = ({ open, onCancel, onSuccess, documentId, documentType }) => {
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
            // @ts-ignore
            const { GetAllUsers } = await import('../../wailsjs/go/services/UserService');
            const data = await GetAllUsers();
            const allUsers = data || [];
            const filteredUsers = allUsers.filter((u: any) =>
                u.roles?.includes('executor') || u.roles?.includes('clerk')
            );
            setUsers(filteredUsers);
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
            // Create expects (documentID, documentType, content, userIds)
            await Create(documentId, documentType, values.content || '', values.userIds);

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
