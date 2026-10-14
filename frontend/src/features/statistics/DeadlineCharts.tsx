import React from 'react';
import { Col } from 'antd';
import { Column, Line } from '@ant-design/plots';
import { DeadlineRatingTable, ChartCard } from './statisticsShared';

const DeadlineCharts = ({ monthlyConfig, executorConfig, hasMonthlyData, hasExecutorData, ratingItems, year }: any) => <>
  <Col xs={24} xl={10}><ChartCard title={`Контроли по месяцам, ${year} год`} isEmpty={!hasMonthlyData} emptyDescription="Контроли за выбранный год не найдены. Измените период или установите контроль."><Column {...monthlyConfig} /></ChartCard></Col>
  <Col xs={24} xl={10}><ChartCard title="Ежемесячно по ответственным исполнителям" isEmpty={!hasExecutorData} emptyDescription="Нет данных по ответственным исполнителям за выбранный год. Проверьте период или фильтр пользователя."><Line {...executorConfig} /></ChartCard></Col>
  <Col xs={24} xl={4}><DeadlineRatingTable items={ratingItems} /></Col>
</>;

export default DeadlineCharts;
