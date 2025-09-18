# CSGO2 自动交易平台

一个功能全面的Counter-Strike 2饰品自动交易平台，支持市场分析、价格追踪和自动化交易策略。

## 🚀 功能特性

### 核心功能
- **Steam登录认证** - 使用Steam OpenID进行安全登录
- **多平台支持** - 集成Steam市场、BUFF、悠悠有品等主流交易平台
- **实时价格监控** - 自动收集和更新各平台价格数据
- **智能套利分析** - 自动发现跨平台套利机会
- **交易策略引擎** - 支持自定义交易策略和自动执行
- **库存管理** - 统一管理各平台的饰品库存

### 技术特性
- **现代化架构** - Go后端 + React前端 + Python数据服务
- **实时通信** - WebSocket支持实时数据推送
- **响应式设计** - 支持桌面和移动端访问
- **Windows兼容** - 专门优化的Windows构建系统
- **高性能** - 异步数据处理和智能缓存

## 📋 系统要求

### Windows
- Windows 10 或更高版本
- Go 1.19+ 
- Node.js 16+
- Python 3.7+

### Linux/macOS
- Go 1.19+
- Node.js 16+
- Python 3.7+

## 🛠️ 安装和部署

### Windows 快速部署

1. **克隆项目**
   ```bash
   git clone <repository-url>
   cd csgo-trader
   ```

2. **运行构建脚本**
   ```cmd
   build.bat
   ```

3. **配置环境变量**
   - 进入 `build` 目录
   - 编辑 `.env` 文件，配置你的API密钥：
     - `STEAM_API_KEY`: 从 [Steam API Keys](https://steamcommunity.com/dev/apikey) 获取
     - `BUFF_API_KEY`: 从BUFF163账户设置获取
     - `YOUPIN_API_KEY`: 从悠悠有品账户设置获取

4. **启动应用**
   ```cmd
   cd build
   start.bat
   ```

5. **访问应用**
   - 打开浏览器访问 `http://localhost:8080`

### Linux/macOS 部署

1. **克隆项目**
   ```bash
   git clone <repository-url>
   cd csgo-trader
   ```

2. **运行构建脚本**
   ```bash
   ./build.sh
   ```

3. **配置环境变量**
   ```bash
   cd build
   cp .env.example .env
   # 编辑 .env 文件配置API密钥
   ```

4. **启动应用**
   ```bash
   ./start.sh
   ```

## 🎯 使用指南

### 1. 登录系统
- 点击"Steam登录"按钮
- 在Steam页面确认登录
- 系统会自动获取你的Steam信息

### 2. 市场分析
- **价格图表**: 查看各平台的价格走势
- **套利机会**: 自动发现跨平台价格差异
- **热门物品**: 查看价格变动最大的物品

### 3. 设置交易策略
- 创建自定义交易策略
- 设置买入/卖出价格阈值
- 启用自动交易执行

### 4. 库存管理
- 查看Steam库存
- 监控各平台的物品状态
- 管理交易历史

## 🏗️ 项目结构

```
csgo-trader/
├── main.go                 # Go主程序入口
├── internal/               # Go内部包
│   ├── api/               # API路由和处理器
│   ├── config/            # 配置管理
│   ├── database/          # 数据库操作
│   ├── models/            # 数据模型
│   ├── services/          # 业务逻辑服务
│   └── websocket/         # WebSocket处理
├── web/                   # React前端
│   ├── src/
│   │   ├── components/    # React组件
│   │   ├── pages/         # 页面组件
│   │   └── services/      # 前端服务
│   └── package.json
├── python/                # Python数据收集服务
│   ├── collectors/        # 数据收集器
│   ├── database/          # 数据库管理
│   └── main.py           # Python主程序
├── build.bat             # Windows构建脚本
├── build.sh              # Linux/macOS构建脚本
└── README.md
```

## 🔧 API 文档

### 认证接口
- `GET /api/v1/auth/steam/login` - 获取Steam登录URL
- `GET /api/v1/auth/steam/callback` - Steam登录回调
- `POST /api/v1/auth/logout` - 退出登录

### 市场接口
- `GET /api/v1/market/items` - 获取市场物品列表
- `GET /api/v1/market/items/:id/prices` - 获取物品价格
- `GET /api/v1/market/items/:id/chart` - 获取价格图表
- `GET /api/v1/market/arbitrage` - 获取套利机会

### 交易接口
- `GET /api/v1/trading/strategies` - 获取交易策略
- `POST /api/v1/trading/strategies` - 创建交易策略
- `POST /api/v1/trading/buy` - 购买物品
- `POST /api/v1/trading/sell` - 出售物品

## 🔒 安全说明

- 所有API密钥都存储在本地环境变量中
- Steam登录使用官方OpenID协议
- 数据库文件默认存储在本地
- 建议在生产环境中使用HTTPS

## 🐛 故障排除

### 常见问题

1. **Steam API密钥无效**
   - 确保从Steam开发者页面获取有效的API密钥
   - 检查网络连接和防火墙设置

2. **端口被占用**
   - 默认端口是8080，可在.env文件中修改PORT变量
   - 使用 `netstat -an | findstr 8080` 检查端口占用

3. **Python依赖安装失败**
   - 确保Python版本为3.7+
   - 使用 `pip install -r requirements.txt` 手动安装

4. **前端构建失败**
   - 确保Node.js版本为16+
   - 清除npm缓存：`npm cache clean --force`

## 📈 性能优化

- 数据收集间隔可在配置文件中调整
- 使用数据库索引优化查询性能
- WebSocket连接支持自动重连
- 实施请求速率限制防止API限制

## 🤝 贡献

欢迎提交Issue和Pull Request来改进这个项目。

## 📄 许可证

本项目仅供学习和研究使用。使用时请遵守相关平台的服务条款。

## ⚠️ 免责声明

- 本软件仅用于教育和研究目的
- 使用本软件进行交易的风险由用户自行承担
- 请遵守相关平台的使用条款和当地法律法规
- 作者不对使用本软件造成的任何损失承担责任

## 📞 支持

如有问题，请通过以下方式联系：
- 提交GitHub Issue
- 发送邮件至项目维护者

---

**注意**: 在使用本软件前，请确保你已经阅读并理解了所有相关平台的服务条款，并且你的使用行为符合当地法律法规。