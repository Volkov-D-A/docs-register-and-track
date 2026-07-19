import { useState, useCallback, useEffect, useMemo, useRef } from 'react';
import { Button, Popconfirm, Modal, Select, Tag, Switch, Space, Empty, Spin, App } from 'antd';
import { LinkOutlined, DeleteOutlined, PlusOutlined, ApartmentOutlined, UnlockOutlined, LockOutlined } from '@ant-design/icons';
import { LinkDocuments, UnlinkDocument, GetDocumentLinks, models } from '../../types/link';
import { LinkGraph } from './LinkGraph';
import dayjs from 'dayjs';
import { GetList } from '../../../wailsjs/go/services/DocumentQueryService';
import { DOCUMENT_KIND_ADMINISTRATIVE_ORDER, DOCUMENT_KIND_OUTGOING_LETTER, isAdministrativeOrderKind } from '../../constants/documentKinds';
import { getDocumentLinkTypeLabel, getLinkedDocumentColor, getLinkedDocumentLabel } from '../../config/documentLinkConfig';
import { useDocumentKindAccess } from '../../hooks/useDocumentKindAccess';
import { LatestRequest } from '../../utils/latestRequest';

/**
 * Свойства вкладки связей документа.
 */
interface LinksTabProps {
    documentId: string;
    documentKind: string;
}

/**
 * Вкладка для отображения и управления связями документа.
 * Позволяет добавлять новые связи между документами и просматривать граф связей.
 */
export const LinksTab = ({ documentId, documentKind }: LinksTabProps) => {
    const { message } = App.useApp();
    const { hasAction, kinds, ready: accessReady } = useDocumentKindAccess();
    const [links, setLinks] = useState<models.DocumentLink[]>([]);
    const [loading, setLoading] = useState(false);
    const [isModalVisible, setIsModalVisible] = useState(false);
    const [isGraphVisible, setIsGraphVisible] = useState(false);
    const [isGraphMounted, setIsGraphMounted] = useState(false);
    const [isLocked, setIsLocked] = useState(true);
    const [graphRenderKey, setGraphRenderKey] = useState(0);

    // Form state
    const [targetKind, setTargetKind] = useState<string>(
        isAdministrativeOrderKind(documentKind) ? DOCUMENT_KIND_ADMINISTRATIVE_ORDER : DOCUMENT_KIND_OUTGOING_LETTER,
    );
    const [targetId, setTargetId] = useState('');
    const [linkType, setLinkType] = useState('related');

    // Search state
    const [searchTerm, setSearchTerm] = useState('');
    const [searchLoading, setSearchLoading] = useState(false);
    const [targetOptions, setTargetOptions] = useState<{ value: string; label: string }[]>([]);
    const latestSearchRequestRef = useRef(new LatestRequest());
    const latestLinksRequestRef = useRef(new LatestRequest());
    const activeSearchRef = useRef('');
    const activeDocumentRef = useRef(documentId);
    activeSearchRef.current = `${targetKind}\u0000${searchTerm}`;
    activeDocumentRef.current = documentId;
    const canManageLinks = accessReady && hasAction(documentKind, 'link');
    const creatableKinds = useMemo(
        () => accessReady ? kinds.filter((kind) => hasAction(kind.code, 'create')) : [],
        [accessReady, hasAction, kinds],
    );
    const selectableTargetKinds = useMemo(
        () => accessReady ? (creatableKinds.length > 0 ? creatableKinds : kinds) : [],
        [accessReady, creatableKinds, kinds],
    );
    const availableLinkTypes = useMemo(() => {
        if (isAdministrativeOrderKind(documentKind) && targetKind === DOCUMENT_KIND_ADMINISTRATIVE_ORDER) {
            return [
                { value: 'order_amends', label: 'Изменяет/дополняет приказ' },
                { value: 'order_cancels', label: 'Отменяет приказ' },
                { value: 'related', label: 'Связан с...' },
            ];
        }
        return [
            { value: 'reply', label: 'Ответ на...' },
            { value: 'follow_up', label: 'Во исполнение...' },
            { value: 'related', label: 'Связан с...' },
        ];
    }, [documentKind, targetKind]);

    useEffect(() => {
        if (!accessReady || selectableTargetKinds.length === 0) {
            return;
        }
        if (!selectableTargetKinds.some((kind) => kind.code === targetKind)) {
            const preferredKind = isAdministrativeOrderKind(documentKind)
                && selectableTargetKinds.some((kind) => kind.code === DOCUMENT_KIND_ADMINISTRATIVE_ORDER)
                ? DOCUMENT_KIND_ADMINISTRATIVE_ORDER
                : selectableTargetKinds[0].code;
            activeSearchRef.current = `${preferredKind}\u0000`;
            latestSearchRequestRef.current.invalidate();
            setTargetKind(preferredKind);
            setTargetId('');
            setSearchTerm('');
            setTargetOptions([]);
        }
    }, [accessReady, documentKind, selectableTargetKinds, targetKind]);

    useEffect(() => {
        if (!availableLinkTypes.some((item) => item.value === linkType)) {
            setLinkType(availableLinkTypes[0]?.value || 'related');
        }
    }, [availableLinkTypes, linkType]);

    const performSearch = useCallback(async (query: string) => {
        const searchKey = `${targetKind}\u0000${query}`;
        if (activeSearchRef.current !== searchKey) {
            return;
        }
        setSearchLoading(true);
        await latestSearchRequestRef.current.run(
            () => GetList(targetKind, {
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
            } as any),
            {
                isRelevant: () => activeSearchRef.current === searchKey,
                onSuccess: (res) => {
                    const items = res?.items || [];
                    setTargetOptions(items.map((item: any) => {
                        const date = item.registrationDate;
                        const number = item.registrationNumber;
                        const content = item.content || '';
                        return {
                            value: item.id,
                            label: `${number} от ${dayjs(date).format('DD.MM.YYYY')} - ${content}`
                        };
                    }));
                },
                onError: (err) => {
                    console.error("Search error:", err);
                    message.error("Ошибка поиска");
                },
                onSettled: () => setSearchLoading(false),
            },
        );
    }, [message, targetKind]);

    useEffect(() => {
        const latestSearchRequest = latestSearchRequestRef.current;
        latestSearchRequest.invalidate();
        setSearchLoading(false);
        setTargetOptions([]);
        const timeoutId = setTimeout(() => {
            if (accessReady && searchTerm.length >= 2 && targetKind) {
                performSearch(searchTerm);
            } else {
                setTargetOptions([]);
            }
        }, 500);
        return () => {
            clearTimeout(timeoutId);
            latestSearchRequest.invalidate();
        };
    }, [accessReady, performSearch, searchTerm, targetKind]);

    const handleSearch = (val: string) => {
        activeSearchRef.current = `${targetKind}\u0000${val}`;
        latestSearchRequestRef.current.invalidate();
        setSearchLoading(false);
        setTargetOptions([]);
        setSearchTerm(val);
    };

    const handleTargetKindChange = (val: string) => {
        activeSearchRef.current = `${val}\u0000`;
        latestSearchRequestRef.current.invalidate();
        setSearchLoading(false);
        setTargetKind(val);
        setTargetId('');
        setSearchTerm('');
        setTargetOptions([]);
    };

    const openGraph = () => {
        setIsGraphVisible(true);
    };

    const closeGraph = () => {
        setIsGraphVisible(false);
    };

    const handleGraphModalOpenChange = (open: boolean) => {
        if (open) {
            setGraphRenderKey((value) => value + 1);
            setIsGraphMounted(true);
            return;
        }

        setIsGraphMounted(false);
    };

    const fetchLinks = useCallback(async () => {
        if (!documentId || !canManageLinks || activeDocumentRef.current !== documentId) {
            return;
        }
        setLoading(true);
        await latestLinksRequestRef.current.run(
            () => GetDocumentLinks(documentId),
            {
                isRelevant: () => activeDocumentRef.current === documentId,
                onSuccess: (data) => setLinks(data || []),
                onError: (error) => {
                    console.error(error);
                    message.error("Не удалось загрузить связи");
                },
                onSettled: () => setLoading(false),
            },
        );
    }, [canManageLinks, documentId, message]);

    useEffect(() => {
        const latestLinksRequest = latestLinksRequestRef.current;
        if (documentId && canManageLinks) {
            setLinks([]);
            void fetchLinks();
        } else {
            latestLinksRequest.invalidate();
            setLinks([]);
            setLoading(false);
        }

        return () => latestLinksRequest.invalidate();
    }, [canManageLinks, documentId, fetchLinks]);

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
        const otherType = isSource ? item.targetKind : item.sourceKind;
        const otherNumber = isSource ? item.targetNumber : item.sourceNumber;
        const otherSubject = item.targetSubject || ""; // We might only have target subject in my repo query

        const typeLabel = getDocumentLinkTypeLabel(item.linkType);

        const direction = isSource ? "->" : "<-";

        return (
            <div key={item.id} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: 12, borderBottom: '1px solid var(--app-border)' }}>
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
                        <div style={{ color: 'var(--app-text-secondary)', fontSize: 13, wordBreak: 'break-word', whiteSpace: 'pre-wrap' }}>
                            {otherSubject}
                        </div>
                    </div>
                </div>
                <div style={{ flexShrink: 0, marginLeft: 16 }}>
                    {canManageLinks && (
                        <Popconfirm
                            title={`Удалить связь с документом № ${otherNumber || 'без номера'}?`}
                            description="Это действие нельзя отменить. Документы останутся в системе, но связь между ними будет удалена."
                            okText="Удалить связь"
                            cancelText="Отмена"
                            okButtonProps={{ danger: true }}
                            onConfirm={() => handleUnlink(item.id)}
                        >
                            <Button type="text" title="Удалить связь" danger icon={<DeleteOutlined />} />
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
                <Button icon={<ApartmentOutlined />} onClick={openGraph}>
                    Показать граф
                </Button>
            </div>

            {loading ? (
                <div style={{ textAlign: 'center', padding: 20 }}>
                    <Spin />
                </div>
            ) : links.length === 0 ? (
                <Empty
                    description={
                        <span>
                            Связанных документов нет. {canManageLinks ? 'Нажмите "Добавить связь", чтобы связать документ с другим документом.' : 'Связи появятся здесь после добавления пользователем с нужными правами.'}
                        </span>
                    }
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                />
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
                    <Select value={linkType} onChange={setLinkType} style={{ width: '100%' }} options={availableLinkTypes} />

                    <Select value={targetKind || undefined} onChange={handleTargetKindChange} style={{ width: '100%' }}>
                        {selectableTargetKinds.map((kind) => (
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
                            <span style={{ fontSize: '12px', color: 'var(--app-text-muted)' }}>{isLocked ? 'Заблокировано' : 'Разблокировано'}</span>
                        </Space>
                    </div>
                }
                open={isGraphVisible}
                onCancel={closeGraph}
                afterOpenChange={handleGraphModalOpenChange}
                width={1000}
                footer={null}
                destroyOnHidden={true}
                styles={{ body: { height: '600px', padding: 0 } }}
            >
                {isGraphMounted && <LinkGraph key={`${documentId}-${graphRenderKey}`} rootId={documentId} isLocked={isLocked} />}
            </Modal>
        </div>
    );
};
