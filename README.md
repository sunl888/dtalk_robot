# 叮叮通知机器人

## 监听`docker`事件, 过滤出符合条件的事件并且发送通知到叮叮群里。

### Usage
>0. 更改配置文件
>1. 使用`docker-compose up -d`命令来启动一个测试容器
>2. 使用`go run main.go`命令来监听`docker`事件