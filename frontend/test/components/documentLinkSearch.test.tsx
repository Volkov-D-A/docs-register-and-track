import { expect, test, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { LinksTab } from '../../src/components/DocumentLinks/LinksTab';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

const kinds = vi.hoisted(() => [{ code: 'outgoing_letter' }]);
vi.mock('../../src/hooks/useDocumentKindAccess', () => ({
    useDocumentKindAccess: () => ({ hasAction: () => true, kinds, ready: true }),
}));
vi.mock('../../src/components/DocumentLinks/LinkGraph', () => ({ LinkGraph: () => null }));

test('link search requests a bounded cursor result and displays its document options', async () => {
    const search = vi.fn().mockResolvedValue({ items: [{
        id: 'target-1', registrationNumber: 'OUT/1', registrationDate: '2026-10-20', content: 'Ответ на обращение',
    }], hasMore: true, nextCursor: 'unused-by-selector' });
    installWailsMock({
        LinkService: { GetDocumentLinks: vi.fn().mockResolvedValue([]) },
        DocumentQueryService: { GetList: search },
    });
    renderWithApp(<LinksTab documentId="source-1" documentKind="incoming_letter" />);

    fireEvent.click(screen.getByRole('button', { name: /Добавить связь/ }));
    const selectors = await screen.findAllByRole('combobox');
    fireEvent.change(selectors[2], { target: { value: 'Ответ' } });

    await waitFor(() => expect(search).toHaveBeenCalledWith('outgoing_letter', expect.objectContaining({
        search: 'Ответ', pageSize: 20,
    })));
    expect(await screen.findByText('OUT/1 от 20.10.2026 - Ответ на обращение')).toBeInTheDocument();
});
