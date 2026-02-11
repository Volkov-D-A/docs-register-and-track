import React from 'react';
import { Typography, Row, Col, Card, Statistic } from 'antd';
import {
    InboxOutlined,
    SendOutlined,
    CheckSquareOutlined,
    ClockCircleOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';

const { Title } = Typography;

const DashboardPage: React.FC = () => {
    const { user } = useAuthStore();

    return (
        <div>
            <Title level={4}>
                Добро пожаловать, {user?.fullName}!
            </Title>

            <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
                <Col xs={24} sm={12} lg={6}>
                    <Card>
                        <Statistic
                            title="Входящие документы"
                            value={0}
                            prefix={<InboxOutlined />}
                            valueStyle={{ color: '#1677ff' }}
                        />
                    </Card>
                </Col>
                <Col xs={24} sm={12} lg={6}>
                    <Card>
                        <Statistic
                            title="Исходящие документы"
                            value={0}
                            prefix={<SendOutlined />}
                            valueStyle={{ color: '#52c41a' }}
                        />
                    </Card>
                </Col>
                <Col xs={24} sm={12} lg={6}>
                    <Card>
                        <Statistic
                            title="Активные поручения"
                            value={0}
                            prefix={<CheckSquareOutlined />}
                            valueStyle={{ color: '#faad14' }}
                        />
                    </Card>
                </Col>
                <Col xs={24} sm={12} lg={6}>
                    <Card>
                        <Statistic
                            title="Просроченные"
                            value={0}
                            prefix={<ClockCircleOutlined />}
                            valueStyle={{ color: '#ff4d4f' }}
                        />
                    </Card>
                </Col>
            </Row>
        </div>
    );
};

export default DashboardPage;
