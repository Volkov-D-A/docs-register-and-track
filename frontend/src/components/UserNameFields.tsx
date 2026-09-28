import { Checkbox, Form, Input } from 'antd';
import type { CSSProperties } from 'react';

export interface UserNameValues {
    lastName: string;
    firstName: string;
    patronymic: string;
    noPatronymic: boolean;
}

const nameRules = (label: string) => [
    { required: true, whitespace: true, message: `Укажите ${label}` },
    { validator: (_: unknown, value: string) => !value || Array.from(value.trim()).length <= 100
        ? Promise.resolve() : Promise.reject(new Error('Максимум 100 символов')) },
];

export function UserNameFields({ itemStyle }: { itemStyle?: CSSProperties }) {
    const form = Form.useFormInstance();
    const noPatronymic = Form.useWatch('noPatronymic', form) ?? false;
    return <>
        <Form.Item name="lastName" label="Фамилия" rules={nameRules('фамилию')} style={itemStyle}>
            <Input />
        </Form.Item>
        <Form.Item name="firstName" label="Имя" rules={nameRules('имя')} style={itemStyle}>
            <Input />
        </Form.Item>
        <Form.Item name="patronymic" label="Отчество" dependencies={['noPatronymic']}
            rules={noPatronymic ? [] : nameRules('отчество')} style={itemStyle}>
            <Input disabled={noPatronymic} />
        </Form.Item>
        <Form.Item name="noPatronymic" valuePropName="checked" style={itemStyle}>
            <Checkbox onChange={(event) => {
                if (event.target.checked) {
                    form.setFields([{ name: 'patronymic', value: '', errors: [], warnings: [] }]);
                }
            }}>Нет отчества</Checkbox>
        </Form.Item>
    </>;
}
