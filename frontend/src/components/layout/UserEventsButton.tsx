import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Badge, Button, Empty, Popover, Space, Spin, Typography } from 'antd';
import { BellOutlined, CheckOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { dto, models } from '../../../wailsjs/go/models';
import {
    GetUnreadCount,
    GetCurrentUserEvents,
    MarkAllRead,
    MarkRead,
} from '../../../wailsjs/go/services/UserEventService';
import { emitUserEventsReceived, onUserEventsDocumentRead } from '../../events/userEvents';

const { Text } = Typography;
const USER_EVENTS_POLL_INTERVAL_MS = 30000;

type UserEventsButtonProps = {
    onOpenEvent: (event: dto.UserEvent) => void;
};

const eventAccentColor = (eventType: string): string => {
    if (eventType.includes('returned')) {
        return '#ff4d4f';
    }
    if (eventType.includes('finished') || eventType.includes('confirmed')) {
        return '#52c41a';
    }
    if (eventType.includes('completed')) {
        return '#faad14';
    }
    return '#1677ff';
};

const UserEventsButton: React.FC<UserEventsButtonProps> = ({ onOpenEvent }) => {
    const [open, setOpen] = useState(false);
    const [loading, setLoading] = useState(false);
    const [events, setEvents] = useState<dto.UserEvent[]>([]);
    const [unreadCount, setUnreadCount] = useState(0);
    const knownEventIDsRef = useRef<Set<string>>(new Set());

    const rememberLoadedEvents = useCallback((loadedEvents: dto.UserEvent[], notifyNewEvents: boolean) => {
        const loadedEventIDs = loadedEvents
            .map((event) => event.id)
            .filter(Boolean);
        const newUnreadEvents = loadedEvents.filter((event) => (
            event.id && !event.readAt && !knownEventIDsRef.current.has(event.id)
        ));

        knownEventIDsRef.current = new Set([
            ...Array.from(knownEventIDsRef.current),
            ...loadedEventIDs,
        ]);

        if (notifyNewEvents) {
            emitUserEventsReceived(newUnreadEvents);
        }
    }, []);

    const loadEvents = useCallback((showLoading = true, notifyNewEvents = false) => {
        if (showLoading) {
            setLoading(true);
        }
        return Promise.all([
            GetCurrentUserEvents(models.UserEventFilter.createFrom({ page: 1, pageSize: 20 })),
            GetUnreadCount(),
        ])
            .then(([result, count]) => {
                const loadedEvents = result?.items || [];
                rememberLoadedEvents(loadedEvents, notifyNewEvents);
                setEvents(loadedEvents);
                setUnreadCount(count);
            })
            .catch((error) => {
                console.error('GetCurrentUserEvents error:', error);
            })
            .finally(() => {
                if (showLoading) {
                    setLoading(false);
                }
            });
    }, [rememberLoadedEvents]);

    const loadUnreadCount = useCallback(() => (
        GetUnreadCount()
            .then(setUnreadCount)
            .catch((error) => {
                console.error('GetUnreadCount error:', error);
            })
    ), []);

    useEffect(() => {
        void loadEvents(false, false);
    }, [loadEvents]);

    useEffect(() => {
        if (open) {
            void loadEvents(true, true);
        }
    }, [open, loadEvents]);

    useEffect(() => {
        const intervalID = window.setInterval(() => {
            void loadEvents(false, true);
        }, USER_EVENTS_POLL_INTERVAL_MS);

        return () => window.clearInterval(intervalID);
    }, [loadEvents]);

    useEffect(() => onUserEventsDocumentRead((documentId) => {
        const now = new Date().toISOString();
        setEvents((current) => current.map((item) => (
            !item.readAt && item.documentId === documentId
                ? dto.UserEvent.createFrom({ ...item, readAt: now })
                : item
        )));
        void loadUnreadCount();
    }), [loadUnreadCount]);

    const handleOpenEvent = useCallback((event: dto.UserEvent) => {
        const unreadEventsByDocument = events.filter((item) => (
            !item.readAt && item.documentId && item.documentId === event.documentId
        ));
        const eventsToMarkRead = unreadEventsByDocument.length > 0 ? unreadEventsByDocument : [event];

        if (eventsToMarkRead.some((item) => !item.readAt)) {
            const eventIDs = new Set(eventsToMarkRead.map((item) => item.id));
            void Promise.all(eventsToMarkRead.map((item) => MarkRead(item.id)))
                .then(() => {
                    setEvents((current) => current.map((item) => (
                        eventIDs.has(item.id)
                            ? dto.UserEvent.createFrom({ ...item, readAt: new Date().toISOString() })
                            : item
                    )));
                    setUnreadCount((current) => Math.max(0, current - eventIDs.size));
                })
                .catch((error) => {
                    console.error('MarkRead error:', error);
                });
        }
        setOpen(false);
        onOpenEvent(event);
    }, [events, onOpenEvent]);

    const handleMarkAllRead = useCallback(() => {
        void MarkAllRead()
            .then(() => {
                const now = new Date().toISOString();
                setEvents((current) => current.map((item) => (
                    item.readAt ? item : dto.UserEvent.createFrom({ ...item, readAt: now })
                )));
                setUnreadCount(0);
            })
            .catch((error) => {
                console.error('MarkAllRead error:', error);
            });
    }, []);

    const content = useMemo(() => (
        <div style={{ width: 380, maxWidth: 'calc(100vw - 48px)' }}>
            <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 8 }}>
                <Text strong>События</Text>
                <Button
                    size="small"
                    type="text"
                    icon={<CheckOutlined />}
                    disabled={unreadCount === 0}
                    onClick={handleMarkAllRead}
                >
                    Прочитать все
                </Button>
            </Space>

            {loading ? (
                <div style={{ display: 'flex', justifyContent: 'center', padding: '24px 0' }}>
                    <Spin />
                </div>
            ) : events.length === 0 ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Событий нет" />
            ) : (
                <div style={{ maxHeight: 420, overflowY: 'auto' }}>
                    {events.map((event) => (
                        <div
                            key={event.id}
                            role="button"
                            tabIndex={0}
                            onClick={() => handleOpenEvent(event)}
                            onKeyDown={(e) => {
                                if (e.key === 'Enter' || e.key === ' ') {
                                    e.preventDefault();
                                    handleOpenEvent(event);
                                }
                            }}
                            style={{
                                cursor: 'pointer',
                                padding: '12px 0',
                                borderBottom: '1px solid #f0f0f0',
                            }}
                        >
                            <Space align="start" style={{ width: '100%' }}>
                                <span
                                    style={{
                                        width: 8,
                                        height: 8,
                                        marginTop: 7,
                                        borderRadius: 8,
                                        background: event.readAt ? '#d9d9d9' : eventAccentColor(event.eventType),
                                        flex: '0 0 auto',
                                    }}
                                />
                                <Space orientation="vertical" size={2} style={{ minWidth: 0 }}>
                                    <Text strong={!event.readAt} style={{ lineHeight: 1.25 }}>
                                        {event.title}
                                    </Text>
                                    <Text type="secondary" style={{ lineHeight: 1.25 }}>
                                        {event.message}
                                    </Text>
                                    <Text type="secondary" style={{ fontSize: 12 }}>
                                        {dayjs(event.createdAt).format('DD.MM.YYYY HH:mm')}
                                        {event.documentNumber ? ` · ${event.documentNumber}` : ''}
                                    </Text>
                                </Space>
                            </Space>
                        </div>
                    ))}
                </div>
            )}
        </div>
    ), [events, handleMarkAllRead, handleOpenEvent, loading, unreadCount]);

    return (
        <Popover
            open={open}
            onOpenChange={setOpen}
            content={content}
            placement="bottomRight"
            trigger="click"
        >
            <Badge count={unreadCount} size="small">
                <Button icon={<BellOutlined />}>
                    События
                </Button>
            </Badge>
        </Popover>
    );
};

export default UserEventsButton;
