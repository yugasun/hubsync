# hubsync

使用 docker.io 或其他镜像服务来提供（但不限于） gcr.io、k8s.gcr.io、quay.io、ghcr.io 等国外镜像加速下载服务

> 为减少重复请求，合理利用资源，建议提前在 issues 搜索镜像是否已转换过

# 开始使用

## 方案一

要求：严格按照模板规范提交，参考： https://github.com/yugasun/hubsync/issues/2

> 限制：每次提交最多 11 个镜像地址

> Docker 账号有每日镜像拉取限额，请勿滥用

## 方案二

1. 绑定 DockerHub 账号
   在 `Settings`-`Secrets`-`Actions` 选择 `New repository secret` 新建 `DOCKER_USERNAME`（你的 Docker 用户名）
   和 `DOCKER_TOKEN`（你的 Docker 密码） 两个 Secrets

2. 开启 `Settings`-`Options`-`Features` 中的 `Issues` 功能

3. 在 `Issues`-`Labels` 选择 `New label` 依次添加三个 label ：`hubsync`、`success`、`failure`

## 方案三：本地执行

1. 克隆项目：

```shell
git clone https://github.com/yugasun/hubsync
cd hubsync
```

2. 安装依赖：

```shell
go install
```

3. 执行同步：

```shell
go run main.go --username=xxxxxx --password=xxxxxx --content='{ "hubsync": ["hello-world:latest" }'

# 如果需要使用自定义镜像仓库
go run main.go --username=xxxxxx --password=xxxxxx --repository=registry.cn-hangzhou.aliyuncs.com/xxxxxx --content='{ "hubsync": ["hello-world:latest"] }'
```

## License

MIT
