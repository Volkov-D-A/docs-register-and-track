import React from 'react';
import { fireEvent, screen } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import SystemStatisticsTab from '../../src/features/statistics/SystemStatisticsTab';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

test('technical diagnostics load without any object storage operations',async()=>{
 const get=vi.fn().mockResolvedValue({userCount:4,totalDocuments:10,dbSize:'128 MB',generatedAt:'2026-10-14T12:00:00Z',service:{state:'ready'},usage:{},api:{},database:{},outbox:{}});
 installWailsMock({StatisticsService:{GetSystemStatistics:get}});
 renderWithApp(<SystemStatisticsTab/>);
 expect(await screen.findByText('128 MB')).toBeInTheDocument();
 expect(screen.queryByText('Файлы в хранилище')).not.toBeInTheDocument();
 expect(screen.queryByText('Размер хранилища')).not.toBeInTheDocument();
 fireEvent.click(screen.getByRole('button',{name:/Обновить данные/}));
 expect(get).toHaveBeenCalledTimes(2);
});
