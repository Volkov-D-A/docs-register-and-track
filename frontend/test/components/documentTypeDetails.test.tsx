import { expect, test } from 'vitest';
import { screen } from '@testing-library/react';
import IncomingDocumentDetails from '../../src/components/documentDetails/IncomingDocumentDetails';
import OutgoingDocumentDetails from '../../src/components/documentDetails/OutgoingDocumentDetails';
import { renderWithApp } from '../componentTestUtils';

test.each([
    ['incoming', IncomingDocumentDetails],
    ['outgoing', OutgoingDocumentDetails],
] as const)('%s card shows the document type from the current response', (_kind, Details) => {
    renderWithApp(<Details doc={{
        documentTypeId: 'Договор',
        incomingDate: '2026-10-21',
        outgoingDate: '2026-10-21',
        createdAt: '2026-10-21T10:00:00Z',
    }} />);

    expect(screen.getByText('Договор')).toBeInTheDocument();
});
