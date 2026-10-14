import React from 'react';
import { expect, test, vi } from 'vitest';
import { fireEvent, screen, within } from '@testing-library/react';
import { dto } from '../../wailsjs/go/models';
import WorkspaceEventTimeline from '../../src/components/WorkspaceEventTimeline';
import { renderWithApp } from '../componentTestUtils';

const currentTime = new Date('2026-09-28T15:00:00');

test('event timeline shows document details, relative time and semantic icons and opens documents', () => {
    const events = [
        { id: 'assigned', eventType: 'deadline_control_create', entityType: 'deadline_control', createdAt: '2026-09-28T12:47:00',
            title: 'Контроль срока', message: 'Петров П.П. назначил вас исполнителем', documentKind: 'incoming_letter', documentNumber: '238', documentDate: '2026-09-24T00:00:00' },
        { id: 'accepted', eventType: 'deadline_control_complete', entityType: 'deadline_control', createdAt: '2026-09-28T09:41:00',
            title: 'Контроль окончательно исполнен', message: '', documentKind: 'citizen_appeal', documentNumber: '531', documentDate: '2026-09-21T00:00:00' },
        { id: 'stopped', eventType: 'deadline_series_stopped', entityType: 'deadline_series', createdAt: '2026-09-27T17:30:00',
            title: 'Регулярный контроль остановлен', message: '', documentKind: 'administrative_order', documentNumber: '47', readAt: '2026-09-28T08:00:00' },
        { id: 'updated', eventType: 'deadline_control_update', entityType: 'deadline_control', createdAt: '2026-09-26T18:00:00',
            title: 'Контроль изменён', message: '', documentKind: 'outgoing_letter', documentNumber: '' },
    ].map((event) => dto.UserEvent.createFrom({ ...event, documentId: `doc-${event.id}` }));
    const openDocument = vi.fn();
    renderWithApp(<WorkspaceEventTimeline events={events} currentTime={currentTime} onOpenDocument={openDocument} />);
    const timeline = screen.getByRole('list', { name: 'Персональные события' });
    const rows = within(timeline).getAllByRole('listitem');
    expect(rows.map((row) => row.querySelector('time')?.textContent)).toEqual(['12:47', '09:41', 'Вчера', '26.09.2026']);
    expect(rows[0]).toHaveClass('workspace-event--red');
    expect(rows[1]).toHaveClass('workspace-event--green');
    expect(rows[2]).toHaveClass('workspace-event--red', 'workspace-event--read');
    expect(rows[3]).toHaveClass('workspace-event--slate');
    expect(screen.getByText('Входящий № 238 от 24.09.2026')).toBeInTheDocument();
    expect(screen.getByText('Обращение № 531 от 21.09.2026')).toBeInTheDocument();
    expect(screen.getByText('Приказ № 47')).toBeInTheDocument();
    expect(screen.getByText('Исходящий без номера')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Петров П.П. назначил/ }));
    expect(openDocument).toHaveBeenCalledWith('doc-assigned', 'incoming_letter');
});

test('empty timeline shows its empty state', () => {
    renderWithApp(<WorkspaceEventTimeline events={[]} currentTime={currentTime} onOpenDocument={vi.fn()} />);
    expect(screen.getByText('Новых событий нет')).toBeInTheDocument();
    expect(screen.queryByRole('list')).not.toBeInTheDocument();
});
