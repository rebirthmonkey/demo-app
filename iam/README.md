# webserver

## Code






## Deploy

### Local

- init the MySQL database
```shell
mysql -h 127.0.0.1 -u root -p < configs/init.sql
```

- init the Redis database
```shell
sudo vim /usr/local/etc/redis.conf  # enable requirepass with your paswword
brew services restart redis
redis-cli -h 127.0.0.1 -p 6379 -a 
select 0
keys *
sadd groupset group1 group2 group3
smembers groupset
```

- run the Go GIN server
```shell
go mod tidy
go run main.go -c configs/config.yaml
```

- build the executable for CentOS
```shell
export GOOS=linux
export GOARCH=amd64
go build -o goapp main.go
```

- `systemclt` service: only for CentOS
```shell
mkdir -p /usr/local/bin/goapp/configs

cp goapp /usr/local/bin/goapp/
chmod +x /usr/local/bin/goapp/goapp
cp configs/config.yaml /usr/local/bin/goapp/configs/

cat >  /etc/systemd/system/goapp.service <<EOF
[Unit]
Description=Go Application Service

[Service]
ExecStart=/bin/bash -c '/usr/local/bin/goapp/goapp -c /usr/local/bin/goapp/configs/config.yaml >> /tmp/goapp.log 2>&1'
WorkingDirectory=/usr/local/bin/
User=root
Restart=always
Type=simple
KillMode=mixed

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable goapp.service
systemctl start goapp.service
systemctl status goapp.service

systemctl stop goapp.service
systemctl restart goapp.service
```

- test
```shell
curl http://127.0.0.1:8888/ 
curl http://127.0.0.1:8888/users 
curl http://127.0.0.1:8888/groups
```

### Tencent Cloud

#### CVM

- init CDB
```shell
mysql -h nj-cdb-7ajy5j0p.sql.tencentcdb.com -P 63939 -u root -p < configs/init.sql
```

- init Redis
```shell
ssh -i ruan2.pem root@119.45.138.73
redis-cli -h 10.0.1.8 -p 6379 -a XXX
# reuse the same Redis commands for init
```

- upload the executable and config files
```shell
scp -i ../10_tencent/95_lab/ruan2.pem goapp root@119.45.138.73:/data
scp -i ../10_tencent/95_lab/ruan2.pem configs/config.yaml root@119.45.138.73:/data
ssh -i ruan2.pem root@119.45.138.73
chmod +x goapp
./goapp -c config.yaml
```

#### 弹性伸缩

在弹性伸缩的“修改启动配置”中添加：

```shell
#!/bin/bash
mkdir -p /usr/local/bin/goapp/configs

wget "https://ruan-1251956900.cos.ap-guangzhou.myqcloud.com/goapp/goapp" -O /usr/local/bin/goapp/goapp
chmod +x /usr/local/bin/goapp/goapp

wget "https://ruan-1251956900.cos.ap-guangzhou.myqcloud.com/goapp/config.yaml" -O /usr/local/bin/goapp/configs/config.yaml

systemctl stop goapp

cat >  /etc/systemd/system/goapp.service <<EOF
[Unit]
Description=Go Application Service

[Service]
ExecStart=/bin/bash -c '/usr/local/bin/goapp/goapp -c /usr/local/bin/goapp/configs/config.yaml >> /tmp/goapp.log 2>&1'
WorkingDirectory=/usr/local/bin/
User=root
Restart=always
Type=simple
KillMode=mixed

[Install]
WantedBy=multi-user.target
EOF

systemctl enable goapp
systemctl daemon-reload
echo "goapp enabled, now starting..."
systemctl start goapp
```

#### Test

```shell
curl http://EIP:8888/hello 
curl http://EIP:8888/users 
curl http://EIP:8888/groups
```