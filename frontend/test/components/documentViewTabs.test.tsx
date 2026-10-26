import { beforeEach, expect, test, vi } from 'vitest';
import { screen } from '@testing-library/react';
import DocumentViewModal from '../../src/components/DocumentViewModal';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

const state = vi.hoisted(() => ({ ready: true, kind: 'incoming_letter' }));

vi.mock('../../src/hooks/useDocumentKindAccess', () => ({
    useDocumentKindAccess: () => ({ hasAction: () => false, kinds: [], loading: false, ready: state.ready }),
}));
vi.mock('../../src/hooks/useDocumentDetails', () => ({
    useDocumentDetails: () => ({
        data: { id: 'document-1', kindCode: state.kind, registrationNumber: '1', incomingLetter: {}, outgoingLetter: {}, citizenAppeal: {}, administrativeOrder: {} },
        loading: false,
        reload: vi.fn(),
    }),
}));
vi.mock('../../src/features/deadlines/DeadlineControls', () => ({ default: () => null }));
vi.mock('../../src/components/DocumentLinks/LinksTab', () => ({ LinksTab: () => null }));
vi.mock('../../src/components/JournalList', () => ({ default: () => null }));
vi.mock('../../src/components/RelatedDocumentModal', () => ({ default: () => null }));
vi.mock('../../src/components/documentDetails/IncomingDocumentDetails', () => ({ default: () => null }));
vi.mock('../../src/components/documentDetails/OutgoingDocumentDetails', () => ({ default: () => null }));
vi.mock('../../src/components/documentDetails/CitizenAppealDetails', () => ({ default: () => null }));
vi.mock('../../src/components/documentDetails/AdministrativeOrderDetails', () => ({ default: () => null }));

beforeEach(() => {
    state.ready = true;
    installWailsMock({ UserEventService: { MarkDocumentRead: vi.fn().mockResolvedValue(undefined) } });
});

test.each(['incoming_letter', 'outgoing_letter', 'citizen_appeal', 'administrative_order'])(
    '%s card retains the common tabs', async (kind) => {
        state.kind = kind;
        renderWithApp(<DocumentViewModal open documentId="document-1" documentKind={kind} onCancel={() => {}} />);

        expect((await screen.findAllByRole('tab')).map((tab) => tab.textContent)).toEqual([
            'Информация', 'Контроль сроков', 'Связи', 'Журнал',
        ]);
    },
);

test('card waits for access readiness before rendering tabs', () => {
    state.ready = false;
    state.kind = 'incoming_letter';
    renderWithApp(<DocumentViewModal open documentId="document-1" documentKind={state.kind} onCancel={() => {}} />);

    expect(screen.queryByRole('tab')).not.toBeInTheDocument();
    expect(document.querySelector('[aria-busy="true"]')).toBeInTheDocument();
});
