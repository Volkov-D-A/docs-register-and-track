import React from 'react';
import { screen } from '@testing-library/react';
import { expect, test } from 'vitest';
import BackupProgress from '../../src/features/settings/BackupProgress';
import { models } from '../../wailsjs/go/models';
import { renderWithApp } from '../componentTestUtils';

test('shows all server stages including a failed transfer and retry', () => {
  renderWithApp(<BackupProgress state="transferring" stages={[
    { state: 'snapshotting', startedAt: '2026-09-14T00:00:00Z' },
    { state: 'staged', startedAt: '2026-09-14T00:01:00Z', error: 'SMB недоступен' },
    { state: 'transferring', startedAt: '2026-09-14T00:02:00Z' },
  ].map(stage => models.BackupStage.createFrom(stage))} />);
  expect(screen.getByText(/Создание снимка/)).toBeInTheDocument();
  expect(screen.getByText('SMB недоступен')).toBeInTheDocument();
  expect(screen.getByText(/Передача на SMB/)).toBeInTheDocument();
});
