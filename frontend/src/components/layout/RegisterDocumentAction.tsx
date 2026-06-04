import React, { useState } from 'react';
import { Button, Modal, Space, Spin } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { DocumentKindMeta } from '../../constants/documentKinds';
import { useRegisterDocumentStore } from '../../store/useRegisterDocumentStore';

type RegisterDocumentActionProps = {
    availableKinds: DocumentKindMeta[];
    loading: boolean;
    onPageChange: (page: string) => void;
};

const RegisterDocumentAction: React.FC<RegisterDocumentActionProps> = ({
    availableKinds,
    loading,
    onPageChange,
}) => {
    const [open, setOpen] = useState(false);

    if (availableKinds.length === 0) {
        return null;
    }

    return (
        <>
            <Button
                type="primary"
                size="large"
                icon={<PlusOutlined />}
                onClick={() => setOpen(true)}
                style={{
                    position: 'fixed',
                    right: 28,
                    bottom: 28,
                    zIndex: 1000,
                    height: 52,
                    borderRadius: 999,
                    paddingInline: 20,
                    boxShadow: '0 10px 24px rgba(24, 144, 255, 0.24)',
                }}
            >
                Зарегистрировать
            </Button>
            <Modal
                title="Выберите вид документа"
                open={open}
                onCancel={() => setOpen(false)}
                footer={null}
            >
                <Space orientation="vertical" style={{ width: '100%' }}>
                    {loading ? (
                        <div style={{ display: 'flex', justifyContent: 'center', padding: '16px 0' }}>
                            <Spin />
                        </div>
                    ) : (
                        availableKinds.map((kind) => (
                            <Button
                                key={kind.code}
                                block
                                size="large"
                                onClick={() => {
                                    useRegisterDocumentStore.getState().requestOpen(kind.code);
                                    setOpen(false);
                                    onPageChange(kind.pageKey);
                                }}
                            >
                                {kind.label}
                            </Button>
                        ))
                    )}
                </Space>
            </Modal>
        </>
    );
};

export default RegisterDocumentAction;
