import React from 'react';
import { Form, Input, InputNumber } from 'antd';

const { TextArea } = Input;

export const ManualRegistrationNumberField = ({ placeholder = 'Введите номер документа' }: { placeholder?: string }) => (
    <Form.Item name="registrationNumber" label="Регистрационный номер" rules={[{ required: true, message: 'Введите номер вручную' }]}>
        <Input placeholder={placeholder} />
    </Form.Item>
);

export const PagesCountField = ({ name = 'pagesCount', label = 'Кол-во листов', required = false, min = 1 }: { name?: string; label?: string; required?: boolean; min?: number }) => (
    <Form.Item name={name} label={label} rules={required ? [{ required: true, message: 'Укажите кол-во' }] : undefined}>
        <InputNumber min={min} style={{ width: '100%' }} />
    </Form.Item>
);

export const DocumentContentField = ({ rows = 3, required = true, whitespace = false, label = 'Содержание' }: { rows?: number; required?: boolean; whitespace?: boolean; label?: string }) => (
    <Form.Item name="content" label={label} rules={required ? [{ required: true, whitespace, message: whitespace ? 'Укажите содержание' : undefined }] : undefined}>
        <TextArea rows={rows} />
    </Form.Item>
);
