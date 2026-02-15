import { useState, useEffect } from 'react';
import { Button, List, Popconfirm, Modal, Select, message, Tabs, Tag, Switch, Space } from 'antd';
import { LinkOutlined, DeleteOutlined, PlusOutlined, ApartmentOutlined, UnlockOutlined, LockOutlined } from '@ant-design/icons';
import { LinkDocuments, UnlinkDocument, GetDocumentLinks, models } from '../../types/link';
import { LinkGraph } from './LinkGraph';
import dayjs from 'dayjs';

interface LinksTabProps {
    documentId: string;
    documentType: 'incoming' | 'outgoing';
    documentNumber: string; // Passed for context
}

export const LinksTab = ({ documentId, documentType, documentNumber }: LinksTabProps) => {
    const [links, setLinks] = useState<models.DocumentLink[]>([]);
    const [loading, setLoading] = useState(false);
    const [isModalVisible, setIsModalVisible] = useState(false);
    const [isGraphVisible, setIsGraphVisible] = useState(false);
    const [isLocked, setIsLocked] = useState(true);

    // Form state
    const [targetType, setTargetType] = useState<'incoming' | 'outgoing'>('outgoing'); // default opposite?
    const [targetId, setTargetId] = useState('');
    const [linkType, setLinkType] = useState('related');

    // Search state
    const [searchTerm, setSearchTerm] = useState('');
    const [searchLoading, setSearchLoading] = useState(false);
    const [targetOptions, setTargetOptions] = useState<{ value: string; label: string }[]>([]);

    useEffect(() => {
        const timeoutId = setTimeout(() => {
            if (searchTerm.length >= 2) {
                performSearch(searchTerm);
            } else {
                setTargetOptions([]);
            }
        }, 500);
        return () => clearTimeout(timeoutId);
    }, [searchTerm, targetType]);

    const performSearch = async (query: string) => {
        setSearchLoading(true);
        try {
            let items: any[] = [];
            if (targetType === 'incoming') {
                const { GetList } = await import('../../../wailsjs/go/services/IncomingDocumentService');
                const res = await GetList({
                    page: 1, pageSize: 20, search: query,
                    nomenclatureId: '', documentTypeId: '', orgId: '',
                    dateFrom: '', dateTo: '', incomingNumber: '', outgoingNumber: '',
                    senderName: '', outgoingDateFrom: '', outgoingDateTo: '',
                    resolution: '', noResolution: false, nomenclatureIds: []
                } as any);
                items = res?.items || [];
            } else {
                const { GetList } = await import('../../../wailsjs/go/services/OutgoingDocumentService');
                const res = await GetList({
                    page: 1, pageSize: 20, search: query,
                    nomenclatureIds: [], documentTypeId: '', orgId: '',
                    dateFrom: '', dateTo: '', outgoingNumber: '', recipientName: ''
                } as any);
                items = res?.items || [];
            }

            const options = items.map((item: any) => {
                const date = targetType === 'incoming' ? item.incomingDate : item.outgoingDate;
                const number = targetType === 'incoming' ? item.incomingNumber : item.outgoingNumber;
                return {
                    value: item.id,
                    label: `${number} от ${dayjs(date).format('DD.MM.YYYY')} - ${item.subject}`
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
            await LinkDocuments(documentId, targetId, documentType, targetType, linkType);
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
        const otherType = isSource ? item.targetType : item.sourceType;
        const otherNumber = isSource ? item.targetNumber : item.sourceNumber;
        const otherSubject = item.targetSubject || ""; // We might only have target subject in my repo query

        // Link type label
        let typeLabel = item.linkType;
        if (item.linkType === 'reply') typeLabel = 'Ответ';
        if (item.linkType === 'follow_up') typeLabel = 'Во исполнение';
        if (item.linkType === 'related') typeLabel = 'Связан';

        const direction = isSource ? "->" : "<-";

        return (
            <List.Item
                actions={[
                    <Popconfirm title="Удалить связь?" onConfirm={() => handleUnlink(item.id)}>
                        <Button type="text" danger icon={<DeleteOutlined />} />
                    </Popconfirm>
                ]}
            >
                <List.Item.Meta
                    avatar={<LinkOutlined />}
                    title={
                        <span>
                            <Tag color="blue">{typeLabel}</Tag>
                            {direction} {otherType === 'incoming' ? 'Входящий' : 'Исходящий'} № {otherNumber || '???'}
                        </span>
                    }
                    description={otherSubject}
                />
            </List.Item>
        );
    };

    return (
        <div>
            <div style={{ marginBottom: 16, display: 'flex', gap: 8 }}>
                <Button type="primary" icon={<PlusOutlined />} onClick={() => setIsModalVisible(true)}>
                    Добавить связь
                </Button>
                <Button icon={<ApartmentOutlined />} onClick={() => setIsGraphVisible(true)}>
                    Показать граф
                </Button>
            </div>

            <List
                loading={loading}
                dataSource={links}
                renderItem={renderLinkItem}
                locale={{ emptyText: 'Нет связанных документов' }}
            />

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

                    <Select value={targetType} onChange={(val) => { setTargetType(val); setTargetId(''); setSearchTerm(''); setTargetOptions([]); }} style={{ width: '100%' }}>
                        <Select.Option value="incoming">Входящий документ</Select.Option>
                        <Select.Option value="outgoing">Исходящий документ</Select.Option>
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
                destroyOnClose={true}
                styles={{ body: { height: '600px', padding: 0 } }}
            >
                {isGraphVisible && <LinkGraph rootId={documentId} isLocked={isLocked} />}
            </Modal>
        </div>
    );
};
