# CSGO2 自动交易系统

## 项目概述
这是一个功能完整的CSGO2饰品自动交易系统，支持市场分析、自动交易策略、多平台集成等功能。

## 主要功能
- 📊 实时市场数据监控和走势分析
- 🔐 Steam登录和令牌验证
- 🛒 多平台交易支持（BUFF、悠悠有品等）
- 📈 类股票K线图展示
- 🤖 自动交易策略引擎
- 💼 库存管理和利润统计

## 技术栈
- **后端**: Go (API服务) + Python (数据采集)
- **前端**: React + TypeScript + ECharts
- **数据库**: PostgreSQL + Redis
- **消息队列**: RabbitMQ
- **容器化**: Docker + Docker Compose

## 快速开始

### 环境要求
- Go 1.21+
- Python 3.10+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+

### 安装步骤

1. 克隆项目
```bash
git clone <repository>
cd csgo2-trading-bot
```

2. 配置环境变量
```bash
cp config/.env.example config/.env
# 编辑 config/.env 填入必要的配置
```

3. 使用Docker Compose启动
```bash
docker-compose up -d
```

4. 访问前端界面
```
http://localhost:3000
```

## 项目结构
```
csgo2-trading-bot/
├── backend/          # Go后端服务
├── data-collector/   # Python数据采集服务
├── frontend/         # React前端应用
├── config/          # 配置文件
├── docker/          # Docker相关文件
└── docs/            # 项目文档
```

## 安全提示
⚠️ **重要**: 
- 请妥善保管Steam账号信息和API密钥
- 建议使用专门的交易账号
- 定期备份数据库
- 监控异常交易行为

## License
MIT