# 建议的命令

## 前端开发（web 目录）

### 依赖管理
```bash
bun install              # 安装依赖
bun update              # 更新依赖
```

### 开发
```bash
bun run dev             # 启动开发服务器（Vite）
bun run preview         # 预览生产构建
```

### 构建
```bash
bun run build           # 生产构建
```

### 代码质量
```bash
bun run lint            # 检查代码格式（Prettier）
bun run lint:fix        # 修复代码格式
bun run eslint          # 运行 ESLint
bun run eslint:fix      # 修复 ESLint 问题
```

### 国际化
```bash
bun run i18n:extract    # 提取翻译键
bun run i18n:status     # 查看翻译状态
bun run i18n:sync       # 同步翻译文件
bun run i18n:lint       # 检查翻译文件
```

## 后端开发（根目录）

### 构建
```bash
make build-backend      # 构建后端二进制文件
go build -ldflags "-X 'github.com/zhongruan0522/new-api/common.Version=$(git rev-parse HEAD)'" -o new-api
```

### 开发
```bash
go run main.go          # 运行开发服务器
make start-backend      # 启动后端开发服务器（后台）
```

### 测试
```bash
go test ./...           # 运行所有测试
go test -v ./...        # 详细输出
go test -cover ./...    # 显示覆盖率
```

### 代码质量
```bash
go fmt ./...            # 格式化代码
go vet ./...            # 检查代码问题
golangci-lint run       # 运行 linter（如果安装）
```

## 完整构建

### 前端 + 后端
```bash
make all                # 构建前端并启动后端
make build-frontend     # 仅构建前端
```

## 环境配置
```bash
cp .env.example .env    # 创建环境配置文件
# 编辑 .env 根据需要修改配置
```

## Docker
```bash
docker-compose up       # 使用 Docker Compose 启动
docker build -t new-api . # 构建 Docker 镜像
```

## 系统工具
```bash
git status              # 查看 git 状态
git log --oneline       # 查看提交历史
git diff                # 查看未暂存的更改
git add .               # 暂存所有更改
git commit -m "message" # 提交更改
```

## 性能分析
```bash
# 启用 pprof（需要设置 ENABLE_PPROF=true）
# 访问 http://localhost:8005/debug/pprof/
```

## 注意事项
- 前端使用 Bun 作为包管理器（不是 npm/yarn/pnpm）
- 后端需要 Go 1.26.2+
- 数据库支持 SQLite（默认）、MySQL、PostgreSQL
- 环境变量配置在 .env 文件中
- 前端构建时需要设置 VITE_REACT_APP_VERSION 环境变量
