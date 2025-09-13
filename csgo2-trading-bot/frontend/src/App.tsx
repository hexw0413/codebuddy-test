import React, { useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { Layout, message } from 'antd';
import { useDispatch, useSelector } from 'react-redux';
import MainLayout from './components/Layout/MainLayout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Market from './pages/Market';
import Trading from './pages/Trading';
import Inventory from './pages/Inventory';
import Strategy from './pages/Strategy';
import Statistics from './pages/Statistics';
import Settings from './pages/Settings';
import ItemDetail from './pages/ItemDetail';
import { RootState } from './store';
import { checkAuth } from './store/slices/authSlice';
import { connectWebSocket, disconnectWebSocket } from './services/websocket';
import './App.css';

const { Content } = Layout;

function App() {
  const dispatch = useDispatch();
  const { isAuthenticated, loading } = useSelector((state: RootState) => state.auth);

  useEffect(() => {
    // 检查认证状态
    dispatch(checkAuth() as any);
  }, [dispatch]);

  useEffect(() => {
    // 连接WebSocket
    if (isAuthenticated) {
      connectWebSocket();
      
      // 配置全局消息
      message.config({
        top: 60,
        duration: 3,
        maxCount: 3,
      });

      return () => {
        disconnectWebSocket();
      };
    }
  }, [isAuthenticated]);

  if (loading) {
    return (
      <div className="app-loading">
        <div className="loading-spinner">加载中...</div>
      </div>
    );
  }

  return (
    <Router>
      <Routes>
        <Route path="/login" element={
          isAuthenticated ? <Navigate to="/dashboard" /> : <Login />
        } />
        
        <Route path="/" element={
          isAuthenticated ? <MainLayout /> : <Navigate to="/login" />
        }>
          <Route index element={<Navigate to="/dashboard" />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="market" element={<Market />} />
          <Route path="market/item/:id" element={<ItemDetail />} />
          <Route path="trading" element={<Trading />} />
          <Route path="inventory" element={<Inventory />} />
          <Route path="strategy" element={<Strategy />} />
          <Route path="statistics" element={<Statistics />} />
          <Route path="settings" element={<Settings />} />
        </Route>
        
        <Route path="*" element={<Navigate to="/dashboard" />} />
      </Routes>
    </Router>
  );
}

export default App;