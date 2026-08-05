import React from 'react';
import { App } from 'antd';
import { render, type RenderOptions } from '@testing-library/react';

type ServiceMap = Record<string, Record<string, (...args: any[]) => any>>;

export const installWailsMock = (services: ServiceMap) => {
  Object.defineProperty(window, 'go', {
    configurable: true,
    value: { services },
  });
};

export const renderWithApp = (ui: React.ReactElement, options?: Omit<RenderOptions, 'wrapper'>) => (
  render(ui, {
    wrapper: ({ children }) => <App>{children}</App>,
    ...options,
  })
);

export type Deferred<T> = {
  promise: Promise<T>;
  resolve: (value: T) => void;
  reject: (error: unknown) => void;
};

export const deferred = <T,>(): Deferred<T> => {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
};
