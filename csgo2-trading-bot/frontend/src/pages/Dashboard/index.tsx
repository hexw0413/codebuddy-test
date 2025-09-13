import React, { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, Space, Table, Tag, Progress, Typography } from 'antd';
import {
  ArrowUpOutlined,
  ArrowDownOutlined,
  DollarOutlined,
  ShoppingCartOutlined,
  RiseOutlined,
  StockOutlined,
} from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import { useDispatch, useSelector } from 'react-redux';
import { RootState } from '../../store';
import { fetchDashboardData } from '../../store/slices/dashboardSlice';
import './Dashboard.css';

const { Title, Text } = Typography;

const Dashboard: React.FC = () => {
  const dispatch = useDispatch();
  const { stats, recentTrades, profitChart, loading } = useSelector(
    (state: RootState) => state.dashboard
  );

  useEffect(() => {
    dispatch(fetchDashboardData() as any);
    
    // 定时刷新
    const interval = setInterval(() => {
      dispatch(fetchDashboardData() as any);
    }, 30000); // 30秒刷新一次

    return () => clearInterval(interval);
  }, [dispatch]);

  // 利润图表配置
  const profitChartOption = {
    title: {
      text: '利润趋势',
      left: 'center',
    },
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        const date = params[0].axisValue;
        const profit = params[0].value;
        return `${date}<br/>利润: ¥${profit.toFixed(2)}`;
      },
    },
    xAxis: {
      type: 'category',
      data: profitChart?.dates || [],
      axisLabel: {
        rotate: 45,
      },
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        formatter: '¥{value}',
      },
    },
    series: [
      {
        data: profitChart?.values || [],
        type: 'line',
        smooth: true,
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(82, 196, 26, 0.8)' },
              { offset: 1, color: 'rgba(82, 196, 26, 0.1)' },
            ],
          },
        },
        lineStyle: {
          color: '#52c41a',
          width: 2,
        },
      },
    ],
    grid: {
      left: '3%',
      right: '4%',
      bottom: '15%',
      containLabel: true,
    },
  };

  // 交易分布饼图
  const tradePieOption = {
    title: {
      text: '交易平台分布',
      left: 'center',
    },
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c} ({d}%)',
    },
    legend: {
      bottom: '5%',
    },
    series: [
      {
        name: '交易量',
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
          borderWidth: 2,
        },
        label: {
          show: false,
          position: 'center',
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 20,
            fontWeight: 'bold',
          },
        },
        labelLine: {
          show: false,
        },
        data: [
          { value: stats?.steamTrades || 0, name: 'Steam' },
          { value: stats?.buffTrades || 0, name: 'BUFF' },
          { value: stats?.youpinTrades || 0, name: '悠悠有品' },
        ],
      },
    ],
    color: ['#1890ff', '#52c41a', '#faad14'],
  };

  // 最近交易表格列
  const tradeColumns = [
    {
      title: '物品',
      dataIndex: 'itemName',
      key: 'itemName',
      ellipsis: true,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => (
        <Tag color={type === 'buy' ? 'green' : 'red'}>
          {type === 'buy' ? '买入' : '卖出'}
        </Tag>
      ),
    },
    {
      title: '价格',
      dataIndex: 'price',
      key: 'price',
      render: (price: number) => `¥${price.toFixed(2)}`,
    },
    {
      title: '平台',
      dataIndex: 'platform',
      key: 'platform',
      render: (platform: string) => (
        <Tag color={
          platform === 'steam' ? 'blue' :
          platform === 'buff' ? 'green' : 'orange'
        }>
          {platform.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: '利润',
      dataIndex: 'profit',
      key: 'profit',
      render: (profit: number) => (
        <Text type={profit > 0 ? 'success' : 'danger'}>
          {profit > 0 ? '+' : ''}¥{profit.toFixed(2)}
        </Text>
      ),
    },
    {
      title: '时间',
      dataIndex: 'time',
      key: 'time',
    },
  ];

  return (
    <div className="dashboard-container">
      <Title level={2}>仪表盘</Title>
      
      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="stats-row">
        <Col xs={24} sm={12} md={6}>
          <Card className="stat-card hover-card">
            <Statistic
              title="今日利润"
              value={stats?.todayProfit || 0}
              precision={2}
              valueStyle={{ color: stats?.todayProfit > 0 ? '#3f8600' : '#cf1322' }}
              prefix={<DollarOutlined />}
              suffix="¥"
            />
            <div className="stat-footer">
              <span>
                {stats?.profitChange > 0 ? (
                  <><ArrowUpOutlined /> 较昨日 +{stats.profitChange}%</>
                ) : (
                  <><ArrowDownOutlined /> 较昨日 {stats?.profitChange}%</>
                )}
              </span>
            </div>
          </Card>
        </Col>
        
        <Col xs={24} sm={12} md={6}>
          <Card className="stat-card hover-card">
            <Statistic
              title="总交易量"
              value={stats?.totalTrades || 0}
              prefix={<ShoppingCartOutlined />}
              suffix="笔"
            />
            <div className="stat-footer">
              <Progress percent={stats?.tradeProgress || 0} size="small" />
            </div>
          </Card>
        </Col>
        
        <Col xs={24} sm={12} md={6}>
          <Card className="stat-card hover-card">
            <Statistic
              title="库存价值"
              value={stats?.inventoryValue || 0}
              precision={2}
              prefix={<RiseOutlined />}
              suffix="¥"
              valueStyle={{ color: '#1890ff' }}
            />
            <div className="stat-footer">
              <span>{stats?.inventoryCount || 0} 件物品</span>
            </div>
          </Card>
        </Col>
        
        <Col xs={24} sm={12} md={6}>
          <Card className="stat-card hover-card">
            <Statistic
              title="活跃策略"
              value={stats?.activeStrategies || 0}
              prefix={<StockOutlined />}
              suffix="个"
            />
            <div className="stat-footer">
              <span>胜率 {stats?.winRate || 0}%</span>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 图表区域 */}
      <Row gutter={[16, 16]} className="chart-row">
        <Col xs={24} lg={16}>
          <Card title="利润趋势" className="chart-card">
            <ReactECharts option={profitChartOption} style={{ height: 400 }} />
          </Card>
        </Col>
        
        <Col xs={24} lg={8}>
          <Card title="交易分布" className="chart-card">
            <ReactECharts option={tradePieOption} style={{ height: 400 }} />
          </Card>
        </Col>
      </Row>

      {/* 最近交易 */}
      <Card title="最近交易" className="trade-card">
        <Table
          columns={tradeColumns}
          dataSource={recentTrades}
          rowKey="id"
          pagination={false}
          loading={loading}
          scroll={{ x: 800 }}
        />
      </Card>
    </div>
  );
};

export default Dashboard;