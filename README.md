# 分布式能源交易系统
项目描述：这是一个用于小型个体发电用户交易生产电量的项目，旨在解决中心化程度高、交易成本高、透明度低等问题。
## 安装步骤
安装依赖：
1.安装go
```bash
wget https://dl.google.com/go/go1.24.3.linux-amd64.tar.gz
tar -zxvf go1.24.3.linux-amd64.tar.gz -C /usr/local
```
2.配置go环境变量
```bash
sudo sh -c 'echo "#GOLANG" >> /etc/profile'
sudo sh -c 'echo "export GOROOT=/usr/local/go" >> /etc/profile'
sudo sh -c 'echo "export PATH=\$PATH:\$GOROOT/bin" >> /etc/profile'
source /etc/profile
```
3.安装docker并进行相应配置
```bash
sudo apt install docker-compose
sudo systemctl start docker
sudo systemctl enable docker
sudo gpasswd -a $USER docker
newgrp docker
```
4.安装jq
```bash
sudo apt-get install jq
```
5.安装apache2
```bash
sudo apt-get update
sudo apt-get install apache2
sudo systemctl start apache2
```
## 运行本项目
**下载**
```bash
git clone https://github.com/fox766/distributed-energy-system.git
```
**复制distributed-energy-system/application/www文件夹内的index.html和user.html到/var/www/html**
```bash
cd {YOUR_PATH}/distributed-energy-system/application/www
sudo cp index.html user.html /var/www/html
```
**运行network(可能需要代理)**
```bash
cd {YOUR_PATH}/distributed-energy-system/blockchain/network
./start.sh
```
**启动后端代码**
```bash
cd {YOUR_PATH}/distributed-energy-system/application/backend
go run main.go
```
**前往浏览器界面**
转到localhost/index.html即为主界面
**关闭network**
```bash
./stop.sh
```

