# 摸鱼助手 (MoYu Assistant)

一个模块化的 Go 桌面应用，支持**系统托盘最小化**，通过 **Go build tags** 在编译时选择功能模块。

## 功能特性

- 🖥️ 原生 GUI 界面 (基于 Fyne)
- 📌 最小化到系统托盘，关闭窗口不退出
- 🧩 模块化架构，编译时自由组合功能
- 🔧 每个模块完全独立，互不影响

## 可用模块

| 模块 | Build Tag | 说明 |
|------|-----------|------|
| 🕐 世界时钟 | `module_clock` | 多时区时钟显示 |
| ✅ 待办事项 | `module_todo` | 任务管理 |
| 🍅 番茄钟 | `module_pomodoro` | 工作/休息计时器 |
| 📝 快捷笔记 | `module_notes` | 快速文本记录 |
| 📄 MaerskCN BillingConvert | `module_billing_convert` | 社保、公积金账单批量上传文件转换 |

## 快速开始

```bash
# 安装依赖
go mod tidy

# 编译所有模块
make build-all

# 只编译指定模块
make build MODULES="module_clock module_pomodoro"

# 不带任何模块编译（空壳）
make build-minimal

# 直接运行
make run

# 交叉编译 Windows 版本
make build-win
```

## MaerskCN BillingConvert

在左侧菜单中打开 **MaerskCN → BillingConvert**，选择账单 Excel 文件后即可转换为批量上传格式。文件和目录选择使用系统原生选择器；输出目录留空时，结果会保存到源文件所在目录。

可配置员工编号列、生效日期、数据起始行、表头行以及“表头=上传代码”映射。员工编号为空或包含非数字字符的源数据行会被跳过。

每次转换会生成一个包含全部结果的 `MaerskLine*.xlsx`，并额外生成：

- `ML1`、`ML2` 等工作表：每个工作表包含表头和最多 5,000 条数据。
- `ML1.txt`、`ML2.txt` 等文本文件：内容与对应的 `ML` 工作表一致。

BillingConvert 的配置通过应用 Preferences 持久化，与应用其他设置使用相同的配置目录。

## 手动编译

```bash
# 编译全部模块
go build -tags "module_clock module_todo module_pomodoro module_notes module_jiggler module_excel_split module_billing_convert" -o moyu-assistant .

# 只要时钟和待办
go build -tags "module_clock module_todo" -o moyu-assistant .

# Windows 编译（隐藏控制台窗口）
GOOS=windows GOARCH=amd64 go build -tags "module_clock module_todo" -ldflags "-H windowsgui" -o moyu-assistant.exe .
```

## 项目结构

```
├── main.go                          # 入口
├── Makefile                         # 构建脚本
├── internal/
│   ├── app/
│   │   ├── app.go                   # 应用初始化、窗口管理
│   │   └── tray.go                  # 系统托盘逻辑
│   ├── imports/                     # 模块导入枢纽（build tag 控制）
│   │   ├── imports.go
│   │   ├── clock.go
│   │   ├── todo.go
│   │   ├── pomodoro.go
│   │   ├── notes.go
│   │   └── billing_convert.go
│   └── module/
│       ├── registry.go              # 模块注册接口
│       ├── clock/clock.go           # //go:build module_clock
│       ├── todo/todo.go             # //go:build module_todo
│       ├── pomodoro/pomodoro.go     # //go:build module_pomodoro
│       ├── notes/notes.go           # //go:build module_notes
│       └── billing_convert/         # //go:build module_billing_convert
```

## 添加新模块

1. 在 `internal/module/` 下创建新包目录
2. 实现 `module.Module` 接口
3. 文件顶部加 `//go:build module_xxx`
4. 在 `init()` 中调用 `module.Register(&YourModule{})`
5. 在 `internal/imports/` 下添加对应的导入文件
6. 在 `Makefile` 的 `ALL_MODULES` 中添加 tag 名

## 系统要求

- Go 1.21+
- Windows / Linux / macOS（GUI 需要对应平台图形库支持）
