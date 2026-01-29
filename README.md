# fundsTask

Cryptocurrency selection analyzer — 资金与币种数据定时任务服务。

**仓库地址：** <https://github.com/cryptoSelect/fundsTask.git>

```bash
git clone https://github.com/cryptoSelect/fundsTask.git
cd fundsTask
```

## 说明

- 依赖数据库与 [cryptoSelect/public](https://github.com/cryptoSelect/public) 公共库。
- 需配置 `config/config.json`（数据库、登录等），可复制 `config/config.example.json` 为 `config/config.json` 后按需修改。运行后执行登录并启动币种信息、资金流向等定时任务。

## 本地运行

```bash
go mod download
# 配置 config/config.json 后
go run main/main.go
```

## Docker

```bash
# 构建（需先准备好 config/config.json）
docker build -t cryptoselect-fundstask .

# 运行（挂载配置目录）
docker run --rm -v $(pwd)/config:/app/config cryptoselect-fundstask
```

## License

MIT
