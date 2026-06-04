import React from 'react';

export const DetailStack: React.FC<{ children: React.ReactNode }> = ({ children }) => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
        {children}
    </div>
);

export const DetailDivider: React.FC = () => (
    <div style={{ height: 1, background: 'var(--app-border)', margin: '4px 0' }} />
);
