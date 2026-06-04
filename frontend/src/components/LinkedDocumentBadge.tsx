import React from 'react';
import { Tag } from 'antd';
import { getDocumentKindShortLabel } from '../constants/documentKinds';
import { getDocumentLinkTypeLabel } from '../config/documentLinkConfig';

type LinkedDocumentBadgeProps = {
    sourceKind: string;
    sourceNumber: string;
    linkType?: string;
    withMargin?: boolean;
};

const LinkedDocumentBadge: React.FC<LinkedDocumentBadgeProps> = ({
    sourceKind,
    sourceNumber,
    linkType,
    withMargin = true,
}) => {
    const content = (
        <Tag color="blue">
            Создание документа, связанного с: {getDocumentKindShortLabel(sourceKind)} №{sourceNumber}
            {linkType ? ` — ${getDocumentLinkTypeLabel(linkType).toLowerCase()}` : ''}
        </Tag>
    );

    if (!withMargin) {
        return content;
    }

    return (
        <div style={{ marginBottom: 16 }}>
            {content}
        </div>
    );
};

export default LinkedDocumentBadge;
