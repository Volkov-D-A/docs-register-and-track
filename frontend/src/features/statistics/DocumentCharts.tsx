import React from 'react';
import { Col } from 'antd';
import { Column, Line } from '@ant-design/plots';
import { ChartCard } from './statisticsShared';

const DocumentCharts = ({ kindConfig, registrarConfig, hasKindData, hasRegistrarData }: any) => <>
  <Col xs={24} xl={12}><ChartCard title="Ежемесячно по видам документов" isEmpty={!hasKindData} emptyDescription="Документы за выбранный год не найдены. Измените период или зарегистрируйте документы."><Column {...kindConfig} /></ChartCard></Col>
  <Col xs={24} xl={12}><ChartCard title="Ежемесячно по зарегистрировавшему пользователю" isEmpty={!hasRegistrarData} emptyDescription="Нет регистраций для выбранного года. Проверьте период или фильтры отчета."><Line {...registrarConfig} /></ChartCard></Col>
</>;

export default DocumentCharts;
