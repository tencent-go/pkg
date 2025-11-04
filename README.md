# Tencent Go Package

通用 Go 语言工具包，提供常用的工具函数和组件。

[![Go Reference](https://pkg.go.dev/badge/github.com/tencent-go/pkg.svg)](https://pkg.go.dev/github.com/tencent-go/pkg)

## 安装

```bash
# 安装最新正式版
go get github.com/tencent-go/pkg@latest

# 安装最新 beta 版
go get github.com/tencent-go/pkg@beta
```

## Environment Variables

| Name               | Description                         | Default |
|--------------------|-------------------------------------|---------|
| NAMESPACE          | k8s命名空間                             | default |
| SERVICE_NAME       | k8s服務名                              |         |
| POD_NAME           | k8s pod唯一標識                         |         |
| INITIALIZING       | true or false                       |         |
| LOG_LEVEL          | debug, info, warn, error            |         |
| APP_NAME           | Deprecation                         |         |
| SERVICE_HOST       |                                     |         |
| NATS_URLS          | nats://host1:4222,nats://host2:4222 |         |
| ETCD_ENDPOINTS     | 127.0.0.1:2379,127.0.0.2:2379       |         |
| JWT_SECRET_KEY     |                                     | None    |
| MONGODB_URI_BASE64 |                                     |         |
| MONGODB_DB_NAME    |                                     |         |
