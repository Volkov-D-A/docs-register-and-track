import React from 'react';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, test, vi } from 'vitest';
import OutgoingPage from '../../src/pages/OutgoingPage';
import { renderWithApp } from '../componentTestUtils';

vi.mock('../../src/components/DocumentKindPage', () => ({
  default: (props: any) => <div>
    <button onClick={props.onOpenRegister}>open registration</button>
    {props.registerModal.open && <div>
      {props.registerModal.content}
      <button onClick={props.registerModal.onCancel}>close registration</button>
    </div>}
  </div>,
}));
vi.mock('../../src/components/LinkedDocumentBadge', () => ({ default: () => null }));
vi.mock('../../src/hooks/useDocumentKindPageAccess', () => ({
  useDocumentKindPageAccess: () => ({
    accessReady: true,
    canCreateCurrentKind: true,
    isExecutorOnly: false,
    filterDisabled: false,
    pageConfig: {
      title: 'Исходящие',
      registerModalTitle: 'Регистрация',
      registerInitialValues: {},
      tableClassName: '',
      buildColumns: () => [],
      getEditModalTitle: () => 'Редактирование',
    },
  }),
}));
vi.mock('../../src/hooks/useDocumentListPage', () => ({
  useDocumentListPage: () => ({
    data: [], loading: false, page: 1, pageSize: 10, setPage: vi.fn(), setPageSize: vi.fn(), setSearch: vi.fn(),
    hasMore: false, canGoBack: false, goToNextPage: vi.fn(), goToPreviousPage: vi.fn(), load: vi.fn(),
    viewDocId: '', viewModalOpen: false, openViewModal: vi.fn(), closeViewModal: vi.fn(),
  }),
}));
vi.mock('../../src/hooks/useNomenclaturesForKind', () => ({ useNomenclaturesForKind: () => [] }));
vi.mock('../../src/hooks/useReferenceSearch', () => ({ useOrganizationSearch: () => ({ options: [], search: vi.fn() }) }));
vi.mock('../../src/hooks/useDocumentRegistrationActions', () => ({
  useDocumentRegistrationActions: () => ({ registerSubmitting: false, editSubmitting: false, registerDocument: vi.fn(), updateDocument: vi.fn() }),
}));
vi.mock('../../src/store/useDraftLinkStore', () => ({
  useDraftLinkStore: () => ({ sourceId: '', sourceKind: '', sourceNumber: '', targetKind: '', clearDraftLink: vi.fn() }),
}));

test('OutgoingPage confirms before discarding a touched registration form', async () => {
  const user = userEvent.setup();
  renderWithApp(<OutgoingPage />);

  await user.click(screen.getByRole('button', { name: 'open registration' }));
  await user.type(screen.getByLabelText('Адресат (ФИО)'), 'Получатель');
  await user.click(screen.getByRole('button', { name: 'close registration' }));

    expect((await screen.findAllByText('Закрыть без сохранения?')).length).toBeGreaterThan(0);
  expect(screen.getByText('Внесенные изменения будут потеряны.')).toBeInTheDocument();
  await user.click(screen.getByRole('button', { name: 'Закрыть' }));
  expect(screen.queryByLabelText('Адресат (ФИО)')).not.toBeInTheDocument();
});
