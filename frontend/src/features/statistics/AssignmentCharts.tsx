import React from 'react';
import { Col } from 'antd';
import { Column, Line } from '@ant-design/plots';
import { AssignmentRatingTable, ChartCard } from './statisticsShared';

const AssignmentCharts = ({ monthlyConfig, executorConfig, hasMonthlyData, hasExecutorData, ratingItems, year }: any) => <>
  <Col xs={24} xl={10}><ChartCard title={`Поручения по месяцам, ${year} год`} isEmpty={!hasMonthlyData} emptyDescription="Поручения за выбранный год не найдены. Измените период или создайте поручения."><Column {...monthlyConfig} /></ChartCard></Col>
  <Col xs={24} xl={10}><ChartCard title="Ежемесячно по ответственным исполнителям" isEmpty={!hasExecutorData} emptyDescription="Нет данных по ответственным исполнителям за выбранный год. Проверьте период или фильтр пользователя."><Line {...executorConfig} /></ChartCard></Col>
  <Col xs={24} xl={4}><AssignmentRatingTable items={ratingItems} /></Col>
</>;

export default AssignmentCharts;
