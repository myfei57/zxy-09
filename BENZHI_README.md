# SignalFlow

SignalFlow 是智能交通信号控制平台：路口注册后按配时方案运行，方案按时段计划
自动切换，多个路口组成协调组按统一周期和相位偏移形成绿波，检测器流量数据参与
配时调整，设备故障自动降级为固定黄闪并在复归后回到运行方案，值班员可手动接管，
接管超时自动交回控制权，全部操作留审计。

## 运行

```bash
go build -mod=vendor -o signalflow ./cmd/signalflow
./signalflow -addr :8080 -data ./data
```

打开 http://localhost:8080/ 查看路口监控，/ui/plans、/ui/groups、/ui/audit
分别查看方案配置、协调组与审计日志页面。

## 测试

```bash
go test -mod=vendor ./...
go vet -mod=vendor ./...
```

## Docker

```bash
bash build_benzhi_docker.sh signalflow linux/amd64
docker run --rm -p 8080:8080 signalflow bash -c 'go run ./cmd/signalflow -addr :8080 -data /tmp/data'
```
