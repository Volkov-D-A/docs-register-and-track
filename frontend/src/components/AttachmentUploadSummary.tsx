import React from 'react';
import { Alert } from 'antd';
import type { dto } from '../../wailsjs/go/models';
import { formatAppError } from '../utils/appError';

export const AttachmentUploadSummary = ({ result }: { result: dto.AttachmentUploadResult }) => {
    const uploaded = result.items.filter((item) => item.attachment).length;
    const failed = result.items.length - uploaded;
    if (!result.items.length) return null;
    return <Alert
        type={failed ? (uploaded ? 'warning' : 'error') : 'success'}
        showIcon
        title={`Загружено: ${uploaded}. Ошибок: ${failed}.`}
        description={<>
            <ul>
                {result.items.map((item, index) => <li key={index}>
                    {item.filename}: {item.error ? formatAppError(item.error, 'Ошибка загрузки') : 'загружен'}
                </li>)}
            </ul>
            {failed > 0 && <span>Перед повторной загрузкой проверьте список файлов: при потере ответа файл мог сохраниться.</span>}
        </>}
    />;
};
