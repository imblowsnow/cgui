## 编译
####  命令行
```shell
go install github.com/imblowsnow/cgui/chromium/cmd/cgui@latest

cgui dev
cgui build
```



## 问题记录
- [ ] `【无法解决】` 多开占用 UserDataDir 目录，可能导致无法启动(chrome限制，UserDataDir只能同一个进程占用)
- [x] 打包生成exe，带图标信息
- [x] 生成绑定ts/js文件

