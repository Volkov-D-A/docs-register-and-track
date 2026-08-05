import React from 'react';
import { App, Button, Form } from 'antd';
import { act, renderHook, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { useDocumentRegistrationActions } from '../../src/hooks/useDocumentRegistrationActions';
import OutgoingLetterDocumentForm from '../../src/components/documentForms/OutgoingLetterDocumentForm';
import { deferred, installWailsMock, renderWithApp } from '../componentTestUtils';

const HookWrapper = ({ children }: { children: React.ReactNode }) => <App>{children}</App>;

beforeEach(() => {
  vi.spyOn(globalThis.crypto, 'randomUUID')
    .mockReturnValueOnce('11111111-1111-4111-8111-111111111111')
    .mockReturnValue('22222222-2222-4222-8222-222222222222');
});

describe('document registration frontend flow', () => {
  test('keeps the idempotency key after failure and rotates it after success', async () => {
    const register = vi.fn()
      .mockRejectedValueOnce(new Error('temporary failure'))
      .mockResolvedValue({ id: 'document-1' });
    installWailsMock({ DocumentRegistrationService: { Register: register, Update: vi.fn() } });
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useDocumentRegistrationActions({
      kindCode: 'outgoing_letter',
      clearDraftLink: vi.fn(),
    }), { wrapper: HookWrapper });

    await act(async () => {
      await result.current.registerDocument({ payload: { content: 'first' }, successMessage: 'saved', onSuccess });
    });
    await act(async () => {
      await result.current.registerDocument({ payload: { content: 'second' }, successMessage: 'saved', onSuccess });
    });

    expect(register).toHaveBeenNthCalledWith(1, 'outgoing_letter', expect.objectContaining({ idempotencyKey: '11111111-1111-4111-8111-111111111111' }));
    expect(register).toHaveBeenNthCalledWith(2, 'outgoing_letter', expect.objectContaining({ idempotencyKey: '11111111-1111-4111-8111-111111111111' }));
    expect(onSuccess).toHaveBeenCalledTimes(1);

    await act(async () => {
      await result.current.registerDocument({ payload: { content: 'third' }, successMessage: 'saved', onSuccess });
    });
    expect(register).toHaveBeenNthCalledWith(3, 'outgoing_letter', expect.objectContaining({ idempotencyKey: '22222222-2222-4222-8222-222222222222' }));
  });

  test('suppresses a second registration while the first is pending', async () => {
    const pending = deferred<any>();
    const register = vi.fn().mockReturnValue(pending.promise);
    installWailsMock({ DocumentRegistrationService: { Register: register, Update: vi.fn() } });
    const { result } = renderHook(() => useDocumentRegistrationActions({
      kindCode: 'outgoing_letter',
      clearDraftLink: vi.fn(),
    }), { wrapper: HookWrapper });
    const options = { payload: { content: 'same' }, successMessage: 'saved', onSuccess: vi.fn() };

    let first!: Promise<void>;
    let second!: Promise<void>;
    act(() => {
      first = result.current.registerDocument(options);
      second = result.current.registerDocument(options);
    });
    await waitFor(() => expect(register).toHaveBeenCalledTimes(1));
    await act(async () => pending.resolve({ id: 'document-1' }));
    await Promise.all([first, second]);
  });

  test('outgoing form enforces required fields and manual numbering', async () => {
    const onFinish = vi.fn();
    const Harness = () => {
      const [form] = Form.useForm();
      return <>
        <OutgoingLetterDocumentForm
          form={form}
          isEdit={false}
          onFinish={onFinish}
          nomenclatures={[]}
          docTypes={[]}
          orgOptionsRecipient={[]}
          selectedRegisterNomenclature={{ numberingMode: 'manual_only' }}
          onRecipientOrgSearch={vi.fn()}
        />
        <Button onClick={() => form.submit()}>submit document</Button>
      </>;
    };
    const user = userEvent.setup();
    renderWithApp(<Harness />);

    expect(screen.getByLabelText(/Регистрационный номер/)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'submit document' }));
    expect(await screen.findByText('Выберите дело')).toBeInTheDocument();
    expect(onFinish).not.toHaveBeenCalled();
  });
});
