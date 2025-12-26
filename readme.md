# xSync - Twitter Media Downloader

[![Go Reference](https://pkg.go.dev/badge/github.com/WangWilly/xSync.svg)](https://pkg.go.dev/github.com/WangWilly/xSync)
[![Go Report Card](https://goreportcard.com/badge/github.com/WangWilly/xSync)](https://goreportcard.com/report/github.com/WangWilly/xSync)
[![Coverage Status](https://coveralls.io/repos/github/WangWilly/xSync/badge.svg?branch=master)](https://coveralls.io/github/WangWilly/xSync?branch=master)
<!-- [![Go](https://github.com/WangWilly/xSync/actions/workflows/go.yml/badge.svg)](https://github.com/WangWilly/xSync/actions/workflows/go.yml) -->
<!-- ![GitHub Release](https://img.shields.io/github/v/release/unkmonster/tmd)  -->
![GitHub License](https://img.shields.io/github/license/WangWilly/xSync?logo=github)

A cross-platform Twitter media downloader. Easily, quickly, safely, and cleanly download users' tweets in bulk from Twitter. Supports manual user specification or batch downloads through lists and user followings. Ready to use out of the box!

## Features

- Download media tweets from specified users (video, img, gif)
- Preserve tweet titles
- Preserve tweet publication dates, set as file modification time
- Batch download by list
- Batch download from followed users
- Preserve list/following structure in the file system
- Synchronize user/list information: name, protected status, etc.
- Record user's previous names
- Avoid duplicate downloads
  - Record user's latest publication time after each job, only fetch tweets from this point onwards next time
  - Send symbolic links to user directories in list directories, regardless of how many lists contain the same user, only one copy of user archive is saved locally
- Avoid duplicate timeline fetching: tweets within any time period will only be fetched from Twitter once, even if these tweets fail to download. Failed downloads will be stored locally for retry or discard
- Avoid duplicate user synchronization (updating user info, fetching timeline, downloading tweets)
- Rate limiting: avoid triggering Twitter API rate limits
- Automatically follow protected users
- Add backup cookies: improve tweet fetching speed and total quantity

## How to use

### Download/Compile

**Direct Download**

Go to [Release](https://github.com/WangWilly/xSync/releases/latest) to select and download the appropriate version

**Build from Source**

```bash
git clone https://github.com/WangWilly/xSync
cd xSync
go build .
```

### Update/Configure Settings

When running the program for the first time, it will ask for the following configuration information. Please fill in the configuration items as required

#### Configuration Items

1. `storeage path`: Storage path (can be non-existent)
2. `auth_token`: Used for login, [how to obtain](https://github.com/WangWilly/xSync/blob/master/doc/help.md#获取-cookie)
3. `ct0`: Used for login, [how to obtain](https://github.com/WangWilly/xSync/blob/master/doc/help.md#获取-cookie)
4. `max_download_routine`: Maximum concurrent download goroutines (if 0, uses default value)

#### Update Configuration

```shell
xSync --conf
```

> **Executing the above command will cause the configuration wizard to run again, which will reconfigure the entire configuration file, not individual configuration items. To modify individual configuration items**, please manually edit `%appdata%/.x_sync/conf.yaml` or `$HOME/.x_sync/conf.yaml`

### Command Instructions

```
xSync --help                 // Display help
xSync --conf                 // Re-run configuration program
xSync --user <user_id>       // Download tweets from user specified by user_id
xSync --user <screen_name>   // Download tweets from user specified by screen_name
xSync --list <list_id>       // Batch download each user in the list specified by list_id
xSync --foll <user_id>       // Batch download each user followed by the user specified by user_id
xSync --foll <screen_name>   // Batch download each user followed by the user specified by screen_name
xSync --auto-follow          // Automatically follow protected users
xSync --no-retry             // Dump only, do not automatically retry failed tweet downloads before program exit
```

> To create symbolic links, the program should be run as administrator on Windows

[Don't know what user_id/list_id/screen_name is?](https://github.com/WangWilly/xSync/blob/master/doc/help.md#%E8%8E%B7%E5%8F%96-list_id-user_id-screen_name)

### Examples

```
xSync --user elonmusk  // Download user with screen_name 'elonmusk'
xSync --user 1234567   // Download user with user_id 1234567
xSync --list 8901234   // Download list with list_id 8901234
xSync --foll 567890    // Download all users followed by user with user_id 567890
```

Recommended approach: run once

```shell
xSync --user elonmusk --user 1234567 --list 8901234 --foll 567890
```

### Setting up Proxy

Specify the proxy server through environment variables before running (skip this step for TUN mode)

```bash
set HTTP_PROXY=url
set HTTPS_PROXY=url
```

Example:
```bash
set HTTP_PROXY=http://127.0.0.1:7890
set HTTPS_PROXY=http://127.0.0.1:7890
xSync --user elonmusk
```

If you are using Windows, use the following commands in PowerShell to set up the proxy:
```powershell
$Env:HTTP_PROXY="http://127.0.0.1:7890"
$Env:HTTPS_PROXY="http://127.0.0.1:7890"
```

### Ignore Users

The program will ignore muted or blocked users by default, so if the list you want to download contains users you don't want to include, you can mute or block them on Twitter

### Adding Extra Cookies

The program dynamically selects from all available cookies one that won't be rate limited to request user tweets, avoiding program blocking due to rate limits on a single cookie.

Create `$HOME/.x_sync/additional_cookies.yaml` or `%appdata%/.x_sync/additional_cookies.yaml` in the following format

```yaml
- auth_token: xxxxxxxxx1
  ct0: xxxxxxxxxxxxxxxxxxxxxxx
- auth_token: xxxxxxxxx2
  ct0: xxxxxxxxxxxxxxxx2
- auth_token: xxxxxxxxxxxxxxxx3
  ct0: xxxxxxxxxxxxxxxxxxxxx3
```
> These added backup cookies are only used to improve tweet fetching rate and total quantity. Determining whether to ignore users and automatically following protected users still uses the main account

## Details

### About Rate Limiting

Twitter API limits requests that are too frequent within a period of time (for example, a certain endpoint only allows 500 requests per 15 minutes, exceeding this number will result in a 429 response). When a certain endpoint is about to reach the rate limit, the program will print a notification and block the goroutine trying to request this endpoint until the quota is refreshed (this takes at most 15 minutes). However, it will not block all goroutines, so messages printed by other goroutines may cover this sleep notification, making it seem like the program is unresponsive. After waiting for the quota to refresh, the program will continue working.

## Community

Telegram: https://t.me/+I4yyM81HaJpkNTll

## Project Structure

> **Note**: This project has been restructured as a monorepo. See [README-MONOREPO.md](README-MONOREPO.md) for detailed information about the new structure and usage.

- **CLI Application**: `cmd/cli/` - Command line interface
- **Server Application**: `cmd/server/` - Web dashboard
- **Quick Start**: Use `./scripts/build.sh` to build both applications
