import React from 'react';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, expect, test, vi } from 'vitest';
import ReportsPage from '../../src/pages/ReportsPage';
import { deferred, installWailsMock, renderWithApp } from '../componentTestUtils';

const events=vi.hoisted(()=>({listeners:[] as Array<(event:{topic:string})=>void>}));
vi.mock('../../src/events/serverEvents',()=>({onServerEvent:(listener:(event:{topic:string})=>void)=>{events.listeners.push(listener);return()=>{events.listeners=events.listeners.filter((item)=>item!==listener);};}}));
const filters={kinds:[],users:[],executors:[],organizations:[],departments:[],nomenclature:[]};
const result={template:'organizations',rows:[{key:'org',name:'Доступная организация',incoming:1,outgoing:0,count:1,metric:0}],summary:{count:1,incoming:1,outgoing:0,uniqueDocuments:1},definition:'Документы',filters:'Доступная область',timezone:'Asia/Yekaterinburg',startDate:'2026-01-01',endDate:'2026-12-31',generatedAt:'2026-10-14T12:00:00Z',version:2};
beforeEach(()=>{events.listeners=[];});
test('reports use the organization timezone and clear results on access changes without generating another report',async()=>{
 const run=vi.fn().mockResolvedValue(result);
 installWailsMock({ReportingService:{GetFilters:vi.fn().mockResolvedValue(filters),Run:run}});
 renderWithApp(<ReportsPage/>);
 fireEvent.click(screen.getByRole('button',{name:/Построить/}));
 expect(await screen.findByText('Доступная организация')).toBeInTheDocument();
 expect(run).toHaveBeenCalledWith(expect.objectContaining({timezone:'Asia/Yekaterinburg',periodBasis:'registration_date'}));
 act(()=>events.listeners.forEach((listener)=>listener({topic:'access-changed'})));
 expect(screen.queryByText('Доступная организация')).not.toBeInTheDocument();
 expect(run).toHaveBeenCalledTimes(1);
});
test('a late report response cannot restore data after access invalidation',async()=>{
 const pending=deferred<typeof result>();const run=vi.fn().mockReturnValue(pending.promise);
 installWailsMock({ReportingService:{GetFilters:vi.fn().mockResolvedValue(filters),Run:run}});
 renderWithApp(<ReportsPage/>);
 fireEvent.click(screen.getByRole('button',{name:/Построить/}));await waitFor(()=>expect(run).toHaveBeenCalledTimes(1));
 act(()=>events.listeners.forEach((listener)=>listener({topic:'resync'})));
 await act(async()=>pending.resolve(result));
 expect(screen.queryByText('Доступная организация')).not.toBeInTheDocument();
});
