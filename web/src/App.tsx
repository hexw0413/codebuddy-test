import React, { useEffect, useState } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { Layout, Menu, Avatar, Dropdown, Button, message } from 'antd';
import {
  DashboardOutlined,
  TradingViewOutlined,
  WalletOutlined,
  SettingOutlined,
  LoginOutlined,
  LogoutOutlined,
  UserOutlined,
  LineChartOutlined,
  ShoppingCartOutlined
} from '@ant-design/icons';
import Dashboard from './pages/Dashboard/Dashboard';
import Market from './pages/Market/Market';
import Trading from './pages/Trading/Trading';
import Inventory from './pages/Inventory/Inventory';
import Strategies from './pages/Strategies/Strategies';
import { authService } from './services/authService';
import { websocketService } from './services/websocketService';
import './App.css';

const { Header, Sider, Content } = Layout;

interface User {
  id: number;
  steam_id: string;
  username: string;
  avatar: string;
}

function App() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [collapsed, setCollapsed] = useState(false);

  useEffect(() => {
    // Initialize WebSocket connection
    websocketService.connect();
    
    // Check if user is logged in
    checkAuthStatus();
    
    return () => {
      websocketService.disconnect();
    };
  }, []);

  const checkAuthStatus = async () => {
    try {
      const userData = await authService.getCurrentUser();
      setUser(userData);
    } catch (error) {
      console.log('User not logged in');
    } finally {
      setLoading(false);
    }
  };

  const handleLogin = async () => {
    try {
      const { login_url } = await authService.getSteamLoginUrl();
      window.location.href = login_url;
    } catch (error) {
      message.error('登录失败');
    }
  };

  const handleLogout = async () => {
    try {
      await authService.logout();
      setUser(null);
      message.success('已退出登录');
    } catch (error) {
      message.error('退出失败');
    }
  };

  const menuItems = [
    {
      key: '/dashboard',
      icon: <DashboardOutlined />,
      label: '仪表盘'
    },
    {
      key: '/market',
      icon: <LineChartOutlined />,
      label: '市场分析'
    },
    {
      key: '/trading',
      icon: <TradingViewOutlined />,
      label: '交易中心'
    },
    {
      key: '/strategies',
      icon: <SettingOutlined />,
      label: '交易策略'
    },
    {
      key: '/inventory',
      icon: <WalletOutlined />,
      label: '库存管理'
    }
  ];

  const userMenuItems = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: '个人资料'
    },
    {
      key: 'settings',
      icon: <SettingOutlined />,
      label: '设置'
    },
    {
      type: 'divider'
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      onClick: handleLogout
    }
  ];

  if (loading) {
    return <div className="loading-container">加载中...</div>;
  }

  if (!user) {
    return (
      <Layout style={{ minHeight: '100vh' }}>
        <Header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div className="header-title">CSGO2 自动交易平台</div>
          <Button type="primary" icon={<LoginOutlined />} onClick={handleLogin}>
            Steam 登录
          </Button>
        </Header>
        <Content style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <div style={{ textAlign: 'center' }}>
            <h1>欢迎使用 CSGO2 自动交易平台</h1>
            <p>请使用 Steam 账号登录以开始使用</p>
            <Button type="primary" size="large" icon={<LoginOutlined />} onClick={handleLogin}>
              Steam 登录
            </Button>
          </div>
        </Content>
      </Layout>
    );
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div className="header-title">CSGO2 自动交易平台</div>
        <div className="user-info">
          <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
            <div style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
              <Avatar src={user.avatar} size="small" />
              <span style={{ marginLeft: 8 }}>{user.username}</span>
            </div>
          </Dropdown>
        </div>
      </Header>
      
      <Layout>
        <Sider
          collapsible
          collapsed={collapsed}
          onCollapse={setCollapsed}
          theme="light"
          width={200}
        >
          <Menu
            mode="inline"
            defaultSelectedKeys={['/dashboard']}
            items={menuItems}
            onClick={({ key }) => {
              window.history.pushState(null, '', key);
              window.location.reload();
            }}
          />
        </Sider>
        
        <Layout>
          <Content>
            <Routes>
              <Route path="/" element={<Navigate to="/dashboard" replace />} />
              <Route path="/dashboard" element={<Dashboard />} />
              <Route path="/market" element={<Market />} />
              <Route path="/trading" element={<Trading />} />
              <Route path="/strategies" element={<Strategies />} />
              <Route path="/inventory" element={<Inventory />} />
            </Routes>
          </Content>
        </Layout>
      </Layout>
    </Layout>
  );
}

export default App;