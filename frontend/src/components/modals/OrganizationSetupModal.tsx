import React from 'react';
import { Form, Input, Modal, Spin } from 'antd';
import type { FormInstance } from 'antd';

type OrganizationSetupModalProps = {
    open: boolean;
    loading: boolean;
    saving: boolean;
    form: FormInstance;
    onSave: () => void;
};

const OrganizationSetupModal: React.FC<OrganizationSetupModalProps> = ({
    open,
    loading,
    saving,
    form,
    onSave,
}) => (
    <Modal
        title="Первичная настройка организации"
        open={open}
        forceRender
        closable={false}
        mask={{ closable: false }}
        keyboard={false}
        cancelButtonProps={{ style: { display: 'none' } }}
        okText="Сохранить"
        confirmLoading={saving}
        onOk={onSave}
        okButtonProps={{ disabled: loading }}
    >
        <Spin spinning={loading}>
            <Form form={form} layout="vertical">
                <Form.Item name="organization_name" label="Название организации" rules={[{ required: true, whitespace: true }]}>
                    <Input placeholder="Полное название организации" />
                </Form.Item>
                <Form.Item name="organization_short_name" label="Краткое название организации" rules={[{ required: true, whitespace: true }]}>
                    <Input placeholder="Краткое название организации" />
                </Form.Item>
            </Form>
        </Spin>
    </Modal>
);

export default OrganizationSetupModal;
