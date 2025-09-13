.PHONY: help build run stop clean test deploy

# 默认目标
help:
	@echo "CSGO2 Trading Bot - 可用命令:"
	@echo ""
	@echo "  make build        - 构建所有Docker镜像"
	@echo "  make run          - 启动所有服务"
	@echo "  make stop         - 停止所有服务"
	@echo "  make clean        - 清理容器和镜像"
	@echo "  make test         - 运行测试"
	@echo "  make deploy       - 部署到生产环境"
	@echo "  make logs         - 查看日志"
	@echo "  make db-migrate   - 运行数据库迁移"
	@echo "  make backup       - 备份数据库"

# 构建Docker镜像
build:
	@echo "构建Docker镜像..."
	docker-compose build

# 启动服务
run:
	@echo "启动服务..."
	docker-compose up -d
	@echo "服务已启动!"
	@echo "前端界面: http://localhost:3000"
	@echo "后端API: http://localhost:8080"
	@echo "Grafana监控: http://localhost:3001 (admin/admin)"

# 停止服务
stop:
	@echo "停止服务..."
	docker-compose down

# 清理
clean:
	@echo "清理容器和镜像..."
	docker-compose down -v
	docker system prune -f

# 运行测试
test:
	@echo "运行后端测试..."
	cd backend && go test ./...
	@echo "运行前端测试..."
	cd frontend && npm test

# 查看日志
logs:
	docker-compose logs -f

# 查看特定服务日志
logs-backend:
	docker-compose logs -f backend

logs-collector:
	docker-compose logs -f data-collector

logs-frontend:
	docker-compose logs -f frontend

# 数据库迁移
db-migrate:
	@echo "运行数据库迁移..."
	docker-compose exec backend ./migrate up

# 数据库备份
backup:
	@echo "备份数据库..."
	@mkdir -p backups
	docker-compose exec postgres pg_dump -U csgo2_user csgo2_trading > backups/backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "备份完成!"

# 恢复数据库
restore:
	@echo "恢复数据库..."
	@read -p "输入备份文件名: " file; \
	docker-compose exec -T postgres psql -U csgo2_user csgo2_trading < backups/$$file
	@echo "恢复完成!"

# 部署到生产环境
deploy:
	@echo "部署到生产环境..."
	@echo "1. 构建生产镜像..."
	docker-compose -f docker-compose.prod.yml build
	@echo "2. 推送镜像到仓库..."
	# docker push your-registry/csgo2-trading-backend:latest
	# docker push your-registry/csgo2-trading-frontend:latest
	# docker push your-registry/csgo2-trading-collector:latest
	@echo "3. 部署到服务器..."
	# ssh your-server "cd /app && docker-compose pull && docker-compose up -d"
	@echo "部署完成!"

# 开发环境
dev:
	@echo "启动开发环境..."
	@echo "启动后端开发服务器..."
	cd backend && go run main.go &
	@echo "启动数据采集服务..."
	cd data-collector && python main.py &
	@echo "启动前端开发服务器..."
	cd frontend && npm start

# 安装依赖
install:
	@echo "安装依赖..."
	cd backend && go mod download
	cd data-collector && pip install -r requirements.txt
	cd frontend && npm install

# 格式化代码
fmt:
	@echo "格式化代码..."
	cd backend && go fmt ./...
	cd frontend && npm run format

# 代码检查
lint:
	@echo "代码检查..."
	cd backend && golangci-lint run
	cd frontend && npm run lint