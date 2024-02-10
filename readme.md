## 编译
####  隐藏控制台
```shell
go build -ldflags "-s -w -H=windowsgui"  
```

## 问题记录
### 多开占用 UserDataDir 目录，可能导致无法启动
