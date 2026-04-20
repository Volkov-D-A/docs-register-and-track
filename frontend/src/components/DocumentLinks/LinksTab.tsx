import { useState, useEffect } from 'react';
import { Button, Popconfirm, Modal, Select, Tag, Switch, Space, Empty, Spin, App } from 'antd';
import { LinkOutlined, DeleteOutlined, PlusOutlined, ApartmentOutlined, UnlockOutlined, LockOutlined } from '@ant-design/icons';
import { LinkDocuments, UnlinkDocument, GetDocumentLinks, models } from '../../types/link';
import { LinkGraph } from './LinkGraph';
import dayjs from 'dayjs';
import { GetList } from '../../../wailsjs/go/services/DocumentQueryService';
import { documentKinds, DOCUMENT_KIND_OUTGOING_LETTER, type RegistrationKind } from '../../constants/documentKinds';
import { getDocumentLinkTypeLabel, getLinkedDocumentColor, getLinkedDocumentLabel } from '../../config/documentLinkConfig';
import { useDocumentKindAccess } from '../../hooks/useDocumentKindAccess';

/**
 * Свойства вкладки связей документа.
 */
interface LinksTabProps {
    documentId: string;
    documentNumber: string; // Passed for context
    documentKind: string;
}

/**
 * Вкладка для отображения и управления связями документа.
 * Позволяет добавлять новые связи между документами и просматривать граф связей.
 */
export const LinksTab = ({ documentId, documentNumber, documentKind }: LinksTabProps) => {
    const { message } = App.useApp();
    const { hasAction, kinds } = useDocumentKindAccess();
    const [links, setLinks] = useState<models.DocumentLink[]>([]);
    const [loading, setLoading] = useState(false);
    const [isModalVisible, setIsModalVisible] = useState(false);
    const [isGraphVisible, setIsGraphVisible] = useState(false);
    const [isLocked, setIsLocked] = useState(true);

    // Form state
    const [targetKind, setTargetKind] = useState<RegistrationKind>(DOCUMENT_KIND_OUTGOING_LETTER);
    const [targetId, setTargetId] = useState('');
    const [linkType, setLinkType] = useState('related');

    // Search state
    const [searchTerm, setSearchTerm] = useState('');
    const [searchLoading, setSearchLoading] = useState(false);
    const [targetOptions, setTargetOptions] = useState<{ value: string; label: string }[]>([]);
    const canManageLinks = hasAction(documentKind, 'link');
    const creatableKinds = kinds.filter((kind) => hasAction(kind.code, 'create'));

    useEffect(() => {
        const timeoutId = setTimeout(() => {
            if (searchTerm.length >= 2) {
                performSearch(searchTerm);
            } else {
                setTargetOptions([]);
            }
        }, 500);
        return () => clearTimeout(timeoutId);
    }, [searchTerm, targetKind]);

    const performSearch = async (query: string) => {
        setSearchLoading(true);
        try {
            const res = await GetList(targetKind, {
                page: 1,
                pageSize: 20,
                search: query,
                nomenclatureIds: [],
                documentTypeId: '',
                orgId: '',
                dateFrom: '',
                dateTo: '',
                incomingNumber: '',
                outgoingNumber: '',
                senderName: '',
                recipientName: '',
                outgoingDateFrom: '',
                outgoingDateTo: '',
                resolution: '',
                noResolution: false,
            } as any);
            const items = res?.items || [];

            const options = items.map((item: any) => {
                const date = item.registrationDate;
                const number = item.registrationNumber;
                const content = item.content || '';
                return {
                    value: item.id,
                    label: `${number} от ${dayjs(date).format('DD.MM.YYYY')} - ${content}`
                };
            });
            setTargetOptions(options);
        } catch (err) {
            console.error("Search error:", err);
            message.error("Ошибка поиска");
        } finally {
            setSearchLoading(false);
        }
    };

    const handleSearch = (val: string) => {
        setSearchTerm(val);
    };

    const fetchLinks = async () => {
        setLoading(true);
        try {
            const data = await GetDocumentLinks(documentId);
            setLinks(data || []);
        } catch (error) {
            console.error(error);
            message.error("Не удалось загрузить связи");
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        if (documentId) {
            fetchLinks();
        }
    }, [documentId]);

    const handleLink = async () => {
        if (!targetId) {
            message.warning("Выберите документ");
            return;
        }
        try {
            await LinkDocuments(documentId, targetId, linkType);
            message.success("Связь создана");
            setIsModalVisible(false);
            fetchLinks();
            setTargetId('');
        } catch (error) {
            console.error(error);
            message.error("Ошибка при создании связи");
        }
    };

    const handleUnlink = async (id: string) => {
        try {
            await UnlinkDocument(id);
            message.success("Связь удалена");
            fetchLinks();
        } catch (error) {
            console.error(error);
            message.error("Ошибка при удалении связи");
        }
    };

    // Determine the "other" document in the link to display
    const renderLinkItem = (item: models.DocumentLink) => {
        const isSource = item.sourceId === documentId;
        const otherId = isSource ? item.targetId : item.sourceId;
        const otherType = isSource ? item.targetKind : item.sourceKind;
        const otherNumber = isSource ? item.targetNumber : item.sourceNumber;
        const otherSubject = item.targetSubject || ""; // We might only have target subject in my repo query

        const typeLabel = getDocumentLinkTypeLabel(item.linkType);

        const direction = isSource ? "->" : "<-";

        return (
            <div key={item.id} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: 12, borderBottom: '1px solid #f0f0f0' }}>
                <div style={{ display: 'flex', alignItems: 'flex-start', gap: 16, flex: 1, minWidth: 0 }}>
                    <div style={{ marginTop: 4 }}>
                        <LinkOutlined style={{ fontSize: 18, color: '#1677ff' }} />
                    </div>
                    <div style={{ minWidth: 0 }}>
                        <div style={{ marginBottom: 4, fontWeight: 500 }}>
                            <Tag color="blue">{typeLabel}</Tag>
                            <Tag color={getLinkedDocumentColor(otherType)} style={{ marginInlineStart: 8 }}>
                                {getLinkedDocumentLabel(otherType)}
                            </Tag>
                            {direction} № {otherNumber || '???'}
                        </div>
                        <div style={{ color: 'rgba(0, 0, 0, 0.45)', fontSize: 13, wordBreak: 'break-word', whiteSpace: 'pre-wrap' }}>
                            {otherSubject}
                        </div>
                    </div>
                </div>
                <div style={{ flexShrink: 0, marginLeft: 16 }}>
                    {canManageLinks && (
                        <Popconfirm title="Удалить связь?" onConfirm={() => handleUnlink(item.id)}>
                            <Button type="text" danger icon={<DeleteOutlined />} />
                        </Popconfirm>
                    )}
                </div>
            </div>
        );
    };

    return (
        <div>
            <div style={{ marginBottom: 16, display: 'flex', gap: 8 }}>
                {canManageLinks && (
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => setIsModalVisible(true)}>
                        Добавить связь
                    </Button>
                )}
                <Button icon={<ApartmentOutlined />} onClick={() => setIsGraphVisible(true)}>
                    Показать граф
                </Button>
            </div>

            {loading ? (
                <div style={{ textAlign: 'center', padding: 20 }}>
                    <Spin />
                </div>
            ) : links.length === 0 ? (
                <Empty description="Нет связанных документов" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            ) : (
                <div style={{ display: 'flex', flexDirection: 'column' }}>
                    {links.map(renderLinkItem)}
                </div>
            )}

            <Modal
                title="Добавить связь"
                open={isModalVisible}
                onOk={handleLink}
                onCancel={() => setIsModalVisible(false)}
            >
                <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                    <Select value={linkType} onChange={setLinkType} style={{ width: '100%' }}>
                        <Select.Option value="reply">Ответ на...</Select.Option>
                        <Select.Option value="follow_up">Во исполнение...</Select.Option>
                        <Select.Option value="related">Связан с...</Select.Option>
                    </Select>

                    <Select value={targetKind} onChange={(val) => { setTargetKind(val); setTargetId(''); setSearchTerm(''); setTargetOptions([]); }} style={{ width: '100%' }}>
                        {(creatableKinds.length > 0 ? creatableKinds : documentKinds).map((kind) => (
                            <Select.Option key={kind.code} value={kind.code}>
                                {getLinkedDocumentLabel(kind.code)} документ
                            </Select.Option>
                        ))}
                    </Select>

                    <Select
                        showSearch
                        value={targetId || undefined}
                        placeholder="Поиск документа (введите номер или содержание)"
                        style={{ width: '100%' }}
                        defaultActiveFirstOption={false}
                        filterOption={false}
                        onSearch={handleSearch}
                        onChange={setTargetId}
                        notFoundContent={searchLoading ? <div style={{ padding: 8, textAlign: 'center' }}>Загрузка...</div> : null}
                        options={targetOptions}
                        loading={searchLoading}
                        allowClear
                    />
                </div>
            </Modal>

            <Modal
                title={
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginRight: 24 }}>
                        <span>Граф связей</span>
                        <Space>
                            <Switch
                                checkedChildren={<LockOutlined />}
                                unCheckedChildren={<UnlockOutlined />}
                                checked={isLocked}
                                onChange={setIsLocked}
                            />
                            <span style={{ fontSize: '12px', color: '#888' }}>{isLocked ? 'Заблокировано' : 'Разблокировано'}</span>
                        </Space>
                    </div>
                }
                open={isGraphVisible}
                onCancel={() => setIsGraphVisible(false)}
                width={1000}
                footer={null}
                destroyOnHidden={true}
                styles={{ body: { height: '600px', padding: 0 } }}
            >
                {isGraphVisible && <LinkGraph rootId={documentId} isLocked={isLocked} />}
            </Modal>
        </div>
    );
};
