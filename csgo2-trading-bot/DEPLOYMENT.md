# CSGO2 Trading Bot 部署指南

## 快速开始

### 1. 环境准备

确保您的系统已安装：
- Docker 20.10+
- Docker Compose 2.0+
- Git
- Make (可选)

### 2. 克隆项目

```bash
git clone <repository-url>
cd csgo2-trading-bot
```

### 3. 配置环境变量

```bash
# 复制环境变量模板
cp config/.env.example config/.env

# 编辑配置文件，填入必要的API密钥
nano config/.env
```

**重要配置项：**
- `STEAM_API_KEY`: Steam Web API密钥
- `BUFF_COOKIE`: BUFF登录Cookie
- `YOUPIN_API_KEY`: 悠悠有品API密钥

### 4. 启动服务

使用Docker Compose：
```bash
docker-compose up -d
```

或使用Makefile：
```bash
make build
make run
```

### 5. 访问服务

- 前端界面: http://localhost:3000
- 后端API: http://localhost:8080
- Grafana监控: http://localhost:3001 (默认: admin/admin)
- RabbitMQ管理: http://localhost:15672 (默认: admin/admin123)

## 生产环境部署

### 1. 服务器要求

- CPU: 4核心以上
- 内存: 8GB以上
- 存储: 50GB SSD
- 系统: Ubuntu 20.04 LTS / CentOS 8

### 2. 安全配置

#### SSL证书配置
```bash
# 安装Certbot
sudo apt-get install certbot

# 获取证书
sudo certbot certonly --standalone -d your-domain.com

# 配置Nginx SSL
cp docker/nginx/nginx.ssl.conf docker/nginx/nginx.conf
```

#### 防火墙配置
```bash
# 开放必要端口
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

### 3. 性能优化

#### 数据库优化
编辑 `docker/postgres/postgresql.conf`:
```conf
shared_buffers = 2GB
effective_cache_size = 6GB
maintenance_work_mem = 512MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
```

#### Redis优化
编辑 `docker/redis/redis.conf`:
```conf
maxmemory 2gb
maxmemory-policy allkeys-lru
save 900 1
save 300 10
save 60 10000
```

### 4. 监控和日志

#### 查看日志
```bash
# 所有服务日志
docker-compose logs -f

# 特定服务日志
docker-compose logs -f backend
docker-compose logs -f data-collector
```

#### Prometheus监控指标
- 访问: http://localhost:9090
- 查询示例:
  - `rate(http_requests_total[5m])` - 请求速率
  - `histogram_quantile(0.95, http_request_duration_seconds)` - 95分位延迟

#### Grafana仪表板
1. 访问 http://localhost:3001
2. 导入仪表板: `docker/grafana/dashboards/`
3. 配置告警规则

### 5. 备份和恢复

#### 自动备份
```bash
# 设置定时备份
crontab -e
# 添加以下行 (每天凌晨3点备份)
0 3 * * * cd /path/to/csgo2-trading-bot && make backup
```

#### 手动备份
```bash
make backup
```

#### 恢复数据
```bash
make restore
# 输入备份文件名
```

## 故障排除

### 常见问题

#### 1. Docker容器无法启动
```bash
# 检查日志
docker-compose logs <service-name>

# 重新构建
docker-compose build --no-cache <service-name>
```

#### 2. 数据库连接失败
```bash
# 检查数据库状态
docker-compose exec postgres psql -U csgo2_user -d csgo2_trading

# 重置数据库
docker-compose down -v
docker-compose up -d postgres
```

#### 3. 前端无法连接后端
- 检查CORS配置
- 确认API_URL环境变量
- 检查防火墙规则

### 性能问题

#### 内存不足
```bash
# 查看内存使用
docker stats

# 限制容器内存
# 在docker-compose.yml中添加:
services:
  backend:
    mem_limit: 1g
```

#### CPU使用率高
```bash
# 查看进程
docker-compose exec <service> top

# 限制CPU
services:
  data-collector:
    cpus: '2.0'
```

## 升级指南

### 1. 备份数据
```bash
make backup
```

### 2. 拉取最新代码
```bash
git pull origin main
```

### 3. 重新构建和部署
```bash
docker-compose down
docker-compose build
docker-compose up -d
```

### 4. 运行迁移
```bash
make db-migrate
```

## 安全建议

1. **定期更新依赖**
   ```bash
   cd backend && go get -u ./...
   cd data-collector && pip install --upgrade -r requirements.txt
   cd frontend && npm update
   ```

2. **使用强密码**
   - 数据库密码
   - Redis密码
   - JWT密钥

3. **限制API访问**
   - 实施速率限制
   - 使用API密钥
   - 启用CORS

4. **监控异常活动**
   - 设置Grafana告警
   - 检查日志异常
   - 监控交易行为

5. **数据加密**
   - 使用SSL/TLS
   - 加密敏感配置
   - 加密数据库备份

## 联系支持

如遇到问题，请：
1. 查看日志文件
2. 检查配置文件
3. 查阅文档
4. 提交Issue

## License

MIT License - 详见LICENSE文件