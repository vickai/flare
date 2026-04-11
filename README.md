# Flare

Challenge all bookmarking apps and websites directories, Aim to Be a best performance monster.

🚧 **Code is being prepared and refactored, commits are slow.**

## Feature

**Simple**, **Fast**, **Lightweight** and super **Easy** to install and use.

- Written in Go (Golang) and a little Modern vanilla Javascript only.
- HTTP stack: [Echo](https://echo.labstack.com/) v5.
- Doesn't depend on any database or any complicated framework.
- Single executable, no dependencies required, good docker support.
- You can choose whether to enable various functions according to your needs: offline mode, weather, editor, account, and so on.

## ScreenShot

TBD

## Documentation

TBD

- Browse automatically generated program documentation:
    - `godoc --http=localhost:8080`



## Directory

```bash
├── build                   build script
├── cmd                     user cli/env parser
├── config                  config for app
│   ├── data                    data for app running
│   ├── define                  define for app launch
│   └── model                   data model for app
├── docker                  docker
├── embed                   resource (assets, template) for web
├── internal
│   ├── auth                user login
│   ├── fn                  fn utils
│   ├── logger              logger
│   ├── misc
│   │   ├── deprecated
│   │   ├── health
│   │   └── redir
│   ├── pages
│   │   ├── editor
│   │   ├── guide
│   │   └── home
│   ├── resources           static resource after minify
│   ├── server
│   ├── settings
│   └── version
└── main.go


# 下载依赖包
go mod tidy

# 直接运行项目
go run main.go

go run main.go --debug

#执行编译脚本
go run build/build.go

```