import React from 'react';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, test, vi } from 'vitest';
import SystemBootstrapGate from '../../src/components/SystemBootstrapGate';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

describe('SystemBootstrapGate', () => {
  test('shows the application only after a successful check', async () => {
    installWailsMock({
      SystemService: {
        GetBootstrapStatus: vi.fn().mockResolvedValue({ state: 'ready', code: 'ready', message: 'Готов' }),
      },
    });

    renderWithApp(<SystemBootstrapGate><div>Форма входа</div></SystemBootstrapGate>);

    expect(await screen.findByText('Форма входа')).toBeInTheDocument();
  });

  test('allows login during maintenance and shows a warning', async () => {
    installWailsMock({
      SystemService: {
        GetBootstrapStatus: vi.fn().mockResolvedValue({ state: 'maintenance', code: 'maintenance', message: 'Обслуживание' }),
      },
    });

    renderWithApp(<SystemBootstrapGate><div>Форма входа</div></SystemBootstrapGate>);

    expect(await screen.findByText('Обслуживание')).toBeInTheDocument();
    expect(screen.getByText('Форма входа')).toBeInTheDocument();
  });

  test('blocks an incompatible client and supports retry', async () => {
    const check = vi.fn()
      .mockResolvedValueOnce({ state: 'client_too_old', code: 'client_too_old', message: 'Обновите приложение' })
      .mockResolvedValueOnce({ state: 'ready', code: 'ready', message: 'Готов' });
    installWailsMock({ SystemService: { GetBootstrapStatus: check } });
    const user = userEvent.setup();

    renderWithApp(<SystemBootstrapGate><div>Форма входа</div></SystemBootstrapGate>);

    expect(await screen.findByText('Обновите приложение')).toBeInTheDocument();
    expect(screen.queryByText('Форма входа')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Повторить проверку' }));
    expect(await screen.findByText('Форма входа')).toBeInTheDocument();
    expect(check).toHaveBeenCalledTimes(2);
  });
});
