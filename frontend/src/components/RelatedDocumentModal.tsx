import React from 'react';
import { Button, Modal, Space, Spin, Typography } from 'antd';
import { DOCUMENT_KIND_ADMINISTRATIVE_ORDER, DocumentKindMeta } from '../constants/documentKinds';

const { Text } = Typography;

type RelatedDocumentModalProps = {
    open: boolean;
    loading: boolean;
    creatableKinds: DocumentKindMeta[];
    sourceIsAdministrativeOrder: boolean;
    onCancel: () => void;
    onCreate: (targetKindCode: string, linkType?: string) => void;
};

const RelatedDocumentModal: React.FC<RelatedDocumentModalProps> = ({
    open,
    loading,
    creatableKinds,
    sourceIsAdministrativeOrder,
    onCancel,
    onCreate,
}) => (
    <Modal
        title="Выберите вид связанного документа"
        open={open}
        onCancel={onCancel}
        footer={null}
        destroyOnHidden
    >
        {loading ? (
            <div style={{ display: 'flex', justifyContent: 'center', padding: '16px 0' }}>
                <Spin />
            </div>
        ) : (
            <Space direction="vertical" style={{ width: '100%' }}>
                {creatableKinds.map((kind) => {
                    const isOrderToOrder = sourceIsAdministrativeOrder
                        && kind.code === DOCUMENT_KIND_ADMINISTRATIVE_ORDER;

                    if (isOrderToOrder) {
                        return (
                            <div
                                key={kind.code}
                                style={{
                                    display: 'flex',
                                    flexDirection: 'column',
                                    gap: 8,
                                    padding: 12,
                                    border: '1px solid var(--app-border)',
                                    borderRadius: 6,
                                }}
                            >
                                <Text strong>{kind.label}</Text>
                                <Button block onClick={() => onCreate(kind.code, 'order_amends')}>
                                    Изменяет/дополняет приказ
                                </Button>
                                <Button block danger onClick={() => onCreate(kind.code, 'order_cancels')}>
                                    Отменяет приказ
                                </Button>
                            </div>
                        );
                    }

                    return (
                        <Button
                            key={kind.code}
                            block
                            size="large"
                            onClick={() => onCreate(kind.code)}
                        >
                            {kind.label}
                        </Button>
                    );
                })}
            </Space>
        )}
    </Modal>
);

export default RelatedDocumentModal;
