import React from 'react';
import { Typography } from 'antd';
import { ReferenceDirectoriesTab } from '../features/settings/ReferenceDirectoriesTab';

const { Title } = Typography;

const ReferencesPage: React.FC = () => (
  <div>
    <Title level={4}>Справочники</Title>
    <ReferenceDirectoriesTab />
  </div>
);

export default ReferencesPage;
