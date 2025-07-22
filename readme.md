# xSync - Twitter Media Downloader

[![Go Reference](https://pkg.go.dev/badge/github.com/WangWilly/xSync.svg)](https://pkg.go.dev/github.com/WangWilly/xSync)
[![Go Report Card](https://goreportcard.com/badge/github.com/WangWilly/xSync)](https://goreportcard.com/report/github.com/WangWilly/xSync)
[![Coverage Status](https://coveralls.io/repos/github/WangWilly/xSync/badge.svg?branch=master)](https://coveralls.io/github/WangWilly/xSync?branch=master)
<!-- [![Go](https://github.com/WangWilly/xSync/actions/workflows/go.yml/badge.svg)](https://github.com/WangWilly/xSync/actions/workflows/go.yml) -->
<!-- ![GitHub Release](https://img.shields.io/github/v/release/unkmonster/tmd)  -->
![GitHub License](https://img.shields.io/github/license/WangWilly/xSync?logo=github)

跨平台的推特媒体下载器。用于轻松，快速，安全，整洁，批量的下载推特上用户的推文。支持手动指定用户或通过列表、用户关注批量下载。开箱即用！

## Feature

- 下载指定用户的媒体推文 (video, img, gif)
- 保留推文标题
- 保留推文发布日期，设置为文件的修改时间
- 以列表为单位批量下载
- 关注中的用户批量下载
- 在文件系统中保留列表/关注结构
- 同步用户/列表信息：名称，是否受保护，等。。。
- 记录用户曾用名
- 避免重复下载
  - 每次工作后记录用户的最新发布时间，下次工作仅从这个时间点开始拉取用户推文
  - 向列表目录发送指向用户目录的符号链接，无论多少列表包含同一用户，本地仅保存一份用户存档
- 避免重复获取时间线：任意一段时间内的推文仅仅会从 twitter 上拉取一次，即使这些推文下载失败。如果下载失败将它们存储到本地，以待重试或丢弃
- 避免重复同步用户（更新用户信息，获取时间线，下载推文）
- 速率限制：避免触发 Twitter API 速率限制
- 自动关注受保护的用户
- 添加备用 cookie：提高推文获取速度和总数量

## How to use

### 下载/编译

**直接下载**

前往 [Release](https://github.com/WangWilly/xSync/releases/latest) 自行选择合适的版本并下载

**自行编译**

```bash
git clone https://github.com/WangWilly/xSync
cd xSync
go build .
```

### 更新/填写配置

第一次运行程序时，程序会询问如下配置信息，请按要求将配置项依次填入

#### 配置项介绍

1. `storeage path`：存储路径(可以不存在)
2. `auth_token`：用于登录，[获取方式](https://github.com/WangWilly/xSync/blob/master/doc/help.md#获取-cookie)
3. `ct0`：用于登录，[获取方式](https://github.com/WangWilly/xSync/blob/master/doc/help.md#获取-cookie)
4. `max_download_routine`：最大并发下载协程数（如果为0取默认值）
5. `database type`：数据库类型，支持 `sqlite` (默认) 或 `postgres`

对于 PostgreSQL，还需要配置：
- `host`：PostgreSQL 服务器地址 (默认: localhost)
- `port`：PostgreSQL 端口 (默认: 5432)
- `username`：数据库用户名
- `password`：数据库密码
- `database name`：数据库名称

#### 更新配置

```shell
xSync --conf
```

> **执行上述命令将导致引导配置程序重新运行，这将重新配置整个配置文件，而不是单独的配置项。单独修改配置项**请至 `%appdata%/.x_sync/conf.yaml` 或 `$HOME/.x_sync/conf.yaml`手动修改

### 数据库支持

xSync 支持两种数据库：

#### SQLite (默认)
- 无需额外安装，开箱即用
- 适合单用户使用
- 数据库文件存储在配置的存储目录中

#### PostgreSQL
- 支持多用户并发访问
- 更好的性能和可扩展性
- 需要单独安装 PostgreSQL 服务器

##### 快速启动 PostgreSQL (使用 Docker)

```bash
# 启动 PostgreSQL 和 pgAdmin
docker-compose -f docker-compose.postgres.yml up -d

# 停止服务
docker-compose -f docker-compose.postgres.yml down
```

默认配置：
- PostgreSQL: `localhost:5432`
- 用户名/密码: `xsync/xsync_password`
- 数据库名: `xsync`
- pgAdmin: `http://localhost:8081` (admin@xsync.local/admin)

##### 环境变量配置

服务器模式下，可以通过环境变量配置数据库：

```bash
# PostgreSQL
export DB_TYPE=postgres
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=xsync
export DB_PASSWORD=xsync_password
export DB_NAME=xsync

# SQLite
export DB_TYPE=sqlite
export DB_PATH=./data/xSync.db
```

### 命令说明

```
xSync --help                 // 显示帮助
xSync --conf                 // 重新运行配置程序
xSync --user <user_id>       // 下载由 user_id 指定的用户的推文
xSync --user <screen_name>   // 下载由 screen_name 指定的用户的推文
xSync --list <list_id>       // 批量下载由 list_id 指定的列表中的每个用户
xSync --foll <user_id>       // 批量下载由 user_id 指定的用户正关注的每个用户
xSync --foll <screen_name>   // 批量下载由 screen_name 指定的用户正关注的每个用户
xSync --auto-follow          // 自动关注受保护的用户
xSync --no-retry             // 仅转储，不在程序退出前自动重试下载失败的推文
```

> 为了创建符号链接，在 Windows 上应该以管理员身份运行程序

[不知道啥是 user_id/list_id/screen_name?](https://github.com/WangWilly/xSync/blob/master/doc/help.md#%E8%8E%B7%E5%8F%96-list_id-user_id-screen_name)

### 示例

```
xSync --user elonmusk  // 下载 screen_name 为 ‘eronmusk’ 的用户
xSync --user 1234567   // 下载 user_id 为 1234567 的用户
xSync --list 8901234   // 下载 list_id 为 8901234 的列表
xSync --foll 567890    // 下载 user_id 为 567890 的用户正关注的所有用户
```

更推荐的做法：一次运行

```shell
xSync --user elonmusk --user 1234567 --list 8901234 --foll 567890
```

### 设置代理

运行前通过环境变量指定代理服务器（TUN 模式跳过这一步）

```bash
set HTTP_PROXY=url
set HTTPS_PROXY=url
```

示例：
```bash
set HTTP_PROXY=http://127.0.0.1:7890
set HTTPS_PROXY=http://127.0.0.1:7890
xSync --user elonmusk
```

如果你使用windows系统，在powershell中使用以下指令设置代理：
```powershell
$Env:HTTP_PROXY="http://127.0.0.1:7890"
$Env:HTTPS_PROXY="http://127.0.0.1:7890"
```

### 忽略用户

程序默认会忽略被静音或被屏蔽的用户，所以当你想要下载的列表中包含你不想包含的用户，可以在推特将他们屏蔽或静音

### 添加额外 cookie

程序动态从所有可用 cookie 中选择一个不会被速率限制的 cookie 请求用户推文，以避免因单一 cookie 的速率限制导致程序被阻塞。

按如下格式创建 `$HOME/.x_sync/additional_cookies.yaml` 或 `%appdata%/.x_sync/additional_cookies.yaml`

```yaml
- auth_token: xxxxxxxxx1
  ct0: xxxxxxxxxxxxxxxxxxxxxxx
- auth_token: xxxxxxxxx2
  ct0: xxxxxxxxxxxxxxxx2
- auth_token: xxxxxxxxxxxxxxxx3
  ct0: xxxxxxxxxxxxxxxxxxxxx3
```
> 这些添加的备用 cookie，仅用来提升获取推文的速率和总量。判断是否忽略用户和自动关注受保护的用户依然使用主账号

## Detail

### 关于速率限制

Twitter API 限制一段时间内过快的请求 （例如某端点每15分钟仅允许请求500次，超出这个次数会以429响应），当某一端点将要达到速率限制程序会打印一条通知并阻塞尝试请求这个端点的协程直到余量刷新（这最多是15分钟），但并不会阻塞所有协程，所以其余协程打印的消息可能将这条休眠通知覆盖让人认为程序无响应了，等待余量刷新程序会继续工作。

## 交流群

tg: https://t.me/+I4yyM81HaJpkNTll

## Project Structure

> **Note**: This project has been restructured as a monorepo. See [README-MONOREPO.md](README-MONOREPO.md) for detailed information about the new structure and usage.

- **CLI Application**: `cmd/cli/` - Command line interface
- **Server Application**: `cmd/server/` - Web dashboard
- **Quick Start**: Use `./scripts/build.sh` to build both applications
