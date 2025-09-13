import React, { useEffect, useState } from 'react';
import {
  Card,
  Table,
  Input,
  Select,
  Button,
  Space,
  Tag,
  Row,
  Col,
  Slider,
  Typography,
  Tooltip,
  Badge,
  message,
} from 'antd';
import {
  SearchOutlined,
  FilterOutlined,
  ReloadOutlined,
  LineChartOutlined,
  ShoppingCartOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import ReactECharts from 'echarts-for-react';
import { useDispatch, useSelector } from 'react-redux';
import { RootState } from '../../store';
import { fetchMarketItems, setFilters } from '../../store/slices/marketSlice';
import './Market.css';

const { Title, Text } = Typography;
const { Option } = Select;

const Market: React.FC = () => {
  const navigate = useNavigate();
  const dispatch = useDispatch();
  const { items, loading, filters, total, trends } = useSelector(
    (state: RootState) => state.market
  );

  const [searchText, setSearchText] = useState('');
  const [priceRange, setPriceRange] = useState<[number, number]>([0, 10000]);
  const [selectedType, setSelectedType] = useState<string>('all');
  const [selectedRarity, setSelectedRarity] = useState<string>('all');
  const [currentPage, setCurrentPage] = useState(1);

  useEffect(() => {
    loadMarketData();
    
    // WebSocket订阅价格更新
    const ws = new WebSocket('ws://localhost:8080/ws');
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'price_update') {
        // 更新价格
        dispatch(updateItemPrice(data.data));
      }
    };

    return () => {
      ws.close();
    };
  }, []);

  const loadMarketData = () => {
    dispatch(fetchMarketItems({
      page: currentPage,
      pageSize: 20,
      ...filters,
    }) as any);
  };

  const handleSearch = () => {
    dispatch(setFilters({
      search: searchText,
      type: selectedType,
      rarity: selectedRarity,
      minPrice: priceRange[0],
      maxPrice: priceRange[1],
    }));
    setCurrentPage(1);
    loadMarketData();
  };

  const handleReset = () => {
    setSearchText('');
    setSelectedType('all');
    setSelectedRarity('all');
    setPriceRange([0, 10000]);
    dispatch(setFilters({}));
    setCurrentPage(1);
    loadMarketData();
  };

  const handleQuickBuy = (item: any) => {
    message.success(`已添加 ${item.name} 到购买列表`);
  };

  // 热门物品图表
  const hotItemsOption = {
    title: {
      text: '热门物品',
      left: 'center',
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow',
      },
    },
    xAxis: {
      type: 'category',
      data: trends?.hotItems?.map((item: any) => item.name.substring(0, 10)) || [],
      axisLabel: {
        rotate: 45,
        interval: 0,
      },
    },
    yAxis: {
      type: 'value',
      name: '交易量',
    },
    series: [
      {
        data: trends?.hotItems?.map((item: any) => item.volume) || [],
        type: 'bar',
        itemStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: '#83bff6' },
              { offset: 0.5, color: '#188df0' },
              { offset: 1, color: '#188df0' },
            ],
          },
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

  const columns = [
    {
      title: '物品',
      dataIndex: 'name',
      key: 'name',
      width: 300,
      render: (text: string, record: any) => (
        <Space>
          <img src={record.iconUrl} alt={text} className="item-icon" />
          <div>
            <div className="item-name">{text}</div>
            <Text type="secondary" className="item-type">
              {record.type} | {record.rarity}
            </Text>
          </div>
        </Space>
      ),
    },
    {
      title: '当前价格',
      dataIndex: 'currentPrice',
      key: 'currentPrice',
      sorter: true,
      render: (price: number, record: any) => (
        <div className="price-cell">
          <Text strong>¥{price.toFixed(2)}</Text>
          {record.priceChange !== 0 && (
            <div className={`price-change ${record.priceChange > 0 ? 'up' : 'down'}`}>
              {record.priceChange > 0 ? '+' : ''}{record.priceChange.toFixed(2)}%
            </div>
          )}
        </div>
      ),
    },
    {
      title: '7日均价',
      dataIndex: 'avgPrice7d',
      key: 'avgPrice7d',
      render: (price: number) => `¥${price.toFixed(2)}`,
    },
    {
      title: '24h成交量',
      dataIndex: 'volume24h',
      key: 'volume24h',
      sorter: true,
      render: (volume: number) => (
        <Badge count={volume} showZero color="#52c41a" />
      ),
    },
    {
      title: '平台',
      dataIndex: 'platforms',
      key: 'platforms',
      render: (platforms: string[]) => (
        <Space>
          {platforms?.map((platform) => (
            <Tag key={platform} color={
              platform === 'steam' ? 'blue' :
              platform === 'buff' ? 'green' : 'orange'
            }>
              {platform.toUpperCase()}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 200,
      render: (_: any, record: any) => (
        <Space>
          <Tooltip title="查看详情">
            <Button
              type="link"
              icon={<LineChartOutlined />}
              onClick={() => navigate(`/market/item/${record.id}`)}
            />
          </Tooltip>
          <Tooltip title="快速购买">
            <Button
              type="primary"
              size="small"
              icon={<ShoppingCartOutlined />}
              onClick={() => handleQuickBuy(record)}
            >
              购买
            </Button>
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div className="market-container">
      <Title level={2}>市场行情</Title>

      {/* 搜索和筛选 */}
      <Card className="filter-card">
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} md={6}>
            <Input
              placeholder="搜索物品名称"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              onPressEnter={handleSearch}
            />
          </Col>
          
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="物品类型"
              value={selectedType}
              onChange={setSelectedType}
              style={{ width: '100%' }}
            >
              <Option value="all">全部类型</Option>
              <Option value="rifle">步枪</Option>
              <Option value="pistol">手枪</Option>
              <Option value="knife">刀具</Option>
              <Option value="gloves">手套</Option>
              <Option value="sticker">贴纸</Option>
            </Select>
          </Col>
          
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="稀有度"
              value={selectedRarity}
              onChange={setSelectedRarity}
              style={{ width: '100%' }}
            >
              <Option value="all">全部稀有度</Option>
              <Option value="contraband">违禁</Option>
              <Option value="covert">隐秘</Option>
              <Option value="classified">保密</Option>
              <Option value="restricted">受限</Option>
              <Option value="milspec">军规</Option>
            </Select>
          </Col>
          
          <Col xs={24} sm={12} md={6}>
            <div className="price-filter">
              <Text>价格区间: ¥{priceRange[0]} - ¥{priceRange[1]}</Text>
              <Slider
                range
                min={0}
                max={10000}
                value={priceRange}
                onChange={setPriceRange}
              />
            </div>
          </Col>
          
          <Col xs={24} sm={12} md={4}>
            <Space>
              <Button
                type="primary"
                icon={<FilterOutlined />}
                onClick={handleSearch}
              >
                筛选
              </Button>
              <Button
                icon={<ReloadOutlined />}
                onClick={handleReset}
              >
                重置
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 市场趋势 */}
      <Row gutter={[16, 16]} className="trends-row">
        <Col xs={24} md={12}>
          <Card title="价格上涨TOP10" className="trend-card">
            <div className="trend-list">
              {trends?.risingItems?.map((item: any, index: number) => (
                <div key={item.id} className="trend-item">
                  <span className="trend-rank">{index + 1}</span>
                  <span className="trend-name">{item.name}</span>
                  <span className="trend-change up">+{item.change}%</span>
                </div>
              ))}
            </div>
          </Card>
        </Col>
        
        <Col xs={24} md={12}>
          <Card className="chart-card">
            <ReactECharts option={hotItemsOption} style={{ height: 300 }} />
          </Card>
        </Col>
      </Row>

      {/* 物品列表 */}
      <Card className="items-card">
        <Table
          columns={columns}
          dataSource={items}
          rowKey="id"
          loading={loading}
          pagination={{
            current: currentPage,
            total: total,
            pageSize: 20,
            onChange: (page) => {
              setCurrentPage(page);
              loadMarketData();
            },
            showSizeChanger: false,
            showTotal: (total) => `共 ${total} 个物品`,
          }}
          scroll={{ x: 1200 }}
        />
      </Card>
    </div>
  );
};

export default Market;