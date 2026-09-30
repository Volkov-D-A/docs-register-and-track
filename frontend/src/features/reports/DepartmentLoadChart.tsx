import React from 'react';
import { Column } from '@ant-design/plots';
import { models } from '../../../wailsjs/go/models';

const DepartmentLoadChart: React.FC<{ rows: models.ReportRow[] }> = ({ rows }) => <Column
  data={rows.map((row) => ({ period: row.period, department: row.name, count: row.count }))}
  xField="period"
  yField="count"
  colorField="department"
  group
  height={320}
  autoFit
/>;

export default DepartmentLoadChart;
