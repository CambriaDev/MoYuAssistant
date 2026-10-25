# 摸鱼助手 (MoYu Assistant)

摸鱼助手是一个使用 Go 和 Fyne 构建的桌面工具箱。应用支持系统托盘、亮色/暗色主题和按需编译模块；关闭主窗口会将应用隐藏到托盘，可从托盘菜单退出。界面优先使用中文字体，找不到可用的中文字体时切换为英文。

## 功能模块

模块通过 Go build tags 单独编译。下表区分已经可用的功能与目前仍处于开发占位状态的模块。

| 模块 | Build tag | 状态 | 功能 |
|------|-----------|------|------|
| MCL Bank | `module_banktool` | 已实现 | 批量将薪资相关 `.xlsx` 文件转换为 `.xls`；可选薪资文件校验银行文件的笔数与金额总和。Windows 上优先尝试使用已安装的 Microsoft Excel，失败时使用 Go 转换器。 |
| BillingConvert | `module_billing_convert` | 已实现 | 将社保、公积金账单转换为批量上传用 Excel 和分片文本文件；可配置员工编号列、日期、行号及表头映射。 |
| MassUpload | `module_excel_split` | 已实现 | 按已支持的工作表类型，将 LonzaCN `LZACN_*.xlsx` 数据填入随项目提供的模板并分别输出。 |
| 假装在线 | `module_jiggler` | 已实现（仅 Windows） | 用户开启后监测键鼠空闲时间，超过配置的秒数时轻微移动鼠标；其他平台显示不支持提示。 |
| 二维码文件传输 | `module_qr_transfer` | 已实现 | 将本地文件拆成二维码帧，在桌面循环播放，供移动设备扫描接收。 |
| 世界时钟 | `module_clock` | 开发中 | 当前为占位界面，时钟功能尚未实现。 |
| 待办事项 | `module_todo` | 开发中 | 当前为占位界面，任务管理功能尚未实现。 |
| 番茄钟 | `module_pomodoro` | 开发中 | 当前为占位界面，计时功能尚未实现。 |
| 快捷笔记 | `module_notes` | 开发中 | 当前为占位界面，笔记功能尚未实现。 |

已实现模块的详细说明：

### MCL Bank

在 **MaerskCN → MCL Bank** 中选择一个或多个 `.xlsx` 文件，或选择文件夹。选择文件夹时会加入其中名称不含 `payroll` 的 `.xlsx` 文件；未选择输入文件时会扫描当前工作目录。转换结果为 `.xls`，默认输出到当前工作目录，也可指定输出目录。可选提供薪资校验文件，比较银行文件与薪资文件的记录数和金额总和。

Windows 上模块会尝试通过 COM 自动化调用本机 Microsoft Excel；如果 Excel 不可用或转换失败，则回退到 Go 转换器。非 Windows 平台使用 Go 转换器。

### BillingConvert

在 **MaerskCN → BillingConvert** 中选择账单 Excel 文件并设置员工编号列、生效日期、数据起始行、表头行和“表头=上传代码”映射。员工编号为空或含非数字字符的行会被跳过。输出目录为空时使用源文件所在目录。

每次转换会生成一个包含全部结果的 `MaerskLine*.xlsx`，并生成 `ML1`、`ML2` 等工作表和对应的 `ML1.txt`、`ML2.txt` 文本文件；每个分片工作表最多包含 5,000 条数据记录。模块配置保存在应用 Preferences 中。

### MassUpload

在 **LonzaCN → MassUpload** 中选择 Excel 文件或文件夹，将输入表格中支持的工作表转换为项目提供的上传模板。支持的源工作表为 `DATA_CHANGE`、`DATA_CHANGE (report only)`、`DATA_CHANGE (cnit)`、`DATA PHF (ViewID=284)` / `DATA_CHANGE (cnphf)`、`DATA SI (ViewId=286)` / `DATA_CHANGE (cnsi)`。文件名需符合 `LZACN_LZACN<三位编号>...` 格式；输出文件名包含处理时间。输出目录可配置并会保存到应用 Preferences。

模板从运行时工作目录下的 `assets/template/LZACN/` 读取。使用此模块时，请确保该模板目录可访问。

### 假装在线

该模块仅在 Windows 上提供实际功能。开启后默认等待 180 秒无键鼠输入，再模拟小幅鼠标移动；超时时间可在界面中调整。功能仅在用户主动开启时运行。

### 二维码文件传输

在 **文件传输 → 二维码文件传输** 中选择本地文件。应用先显示 metadata 二维码，再循环播放数据分片；metadata 包含文件名、大小、分片大小、分片数和 SHA-256 校验值。支持手动切换帧、播放/暂停、拖动进度条、输入二维码序号和点击 20 列分片矩阵跳转。

每片大小可选 300、500、700 或 900 字节，默认 700 字节；播放速度默认 10 FPS，可调整为任意大于 0 的数值。更改分片大小会重新生成传输 ID 和二维码，旧二维码随之失效。应用没有设置文件大小上限，实际可传输大小受内存、二维码识别能力和扫描时间影响；约 10 MiB 是较合适的目标量级。此仓库包含桌面端发送模块，不包含移动端接收网页。

## 构建与运行

需要 Go **1.25.0 或更高版本**。Fyne 桌面应用还需要当前操作系统对应的图形环境和构建工具链。

```bash
# 下载 Go 依赖
go mod download

# 编译并运行全部模块
make build-all
make run-all

# 只编译/运行指定模块
make build MODULES="module_banktool module_qr_transfer"
make run MODULES="module_qr_transfer"

# 编译不含功能模块的空壳
make build-minimal

# 编译 Windows amd64 GUI 程序（隐藏控制台）
make build-win

# 运行所有模块的测试
go test -tags "module_clock module_todo module_pomodoro module_notes module_banktool module_jiggler module_excel_split module_billing_convert module_qr_transfer" ./...
```

`make build` 和 `make run` 默认使用 Makefile 中的全部模块列表；可用 `MODULES` 覆盖。Windows 构建生成 `moyu-assistant.exe`，其他构建目标的文件名为 `moyu-assistant`。

也可直接使用 Go 构建：

```bash
# 编译全部模块
go build -tags "module_clock module_todo module_pomodoro module_notes module_banktool module_jiggler module_excel_split module_billing_convert module_qr_transfer" -o moyu-assistant .

# 只编译二维码文件传输模块
go build -tags "module_qr_transfer" -o moyu-assistant .
```

## 项目结构

```text
.
├── .gitignore
├── Makefile
├── README.md
├── go.mod
├── go.sum
├── main.go
├── assets/
│   └── template/
│       └── LZACN/
│           ├── LZA000001991_Corrections.xlsx
│           ├── LZA000001994_Additional Personal Data (OBSOLETE)_2026_08_03_02_20_44.xlsx
│           ├── LZA000001994_Income Tax_2026_08_03_02_20_43.xlsx
│           ├── LZA000001994_Local one time pay elements_2026_08_03_02_20_44.xlsx
│           ├── LZA000001994_Local recurring pay elements_2026_08_03_02_20_44.xlsx
│           ├── LZA000001994_Public Housing Fund_2026_08_03_02_20_42.xlsx
│           ├── LZA000001994_Social Insurance_2026_08_03_02_20_43.xlsx
│           ├── LZA000001994_Special Payment_2026_08_03_02_20_44.xlsx
│           ├── LZA000001994_Supplementary Insurance_2026_08_03_02_20_44.xlsx
│           └── LZA000001998TPEv005_PayElementUpload.xlsx
└── internal/
    ├── app/
    │   ├── app.go
    │   ├── font.go
    │   ├── menu.go
    │   └── tray.go
    ├── i18n/
    │   └── i18n.go
    ├── imports/
    │   ├── imports.go
    │   ├── banktool.go
    │   ├── billing_convert.go
    │   ├── clock.go
    │   ├── excel_split.go
    │   ├── jiggler.go
    │   ├── notes.go
    │   ├── pomodoro.go
    │   ├── qrtransfer.go
    │   └── todo.go
    └── module/
        ├── header.go
        ├── registry.go
        ├── banktool/
        │   ├── banktool.go
        │   ├── converter_others.go
        │   └── converter_windows.go
        ├── billing_convert/
        │   ├── billing_convert.go
        │   └── billing_convert_test.go
        ├── clock/
        │   └── clock.go
        ├── excel_split/
        │   ├── excel_split.go
        │   └── excel_split_test.go
        ├── jiggler/
        │   ├── jiggler_others.go
        │   └── jiggler_windows.go
        ├── notes/
        │   └── notes.go
        ├── pomodoro/
        │   └── pomodoro.go
        ├── qrtransfer/
        │   ├── matrix.go
        │   ├── protocol.go
        │   ├── protocol_test.go
        │   ├── qrtransfer.go
        │   └── qrtransfer_test.go
        └── todo/
            └── todo.go
```

`main.go` 启动应用并导入模块枢纽。`internal/imports/` 中的 build-tag 条件导入会触发相应模块注册；应用通过统一的 `module.Module` 接口创建模块界面。

## 添加模块

1. 在 `internal/module/` 下创建模块包并实现 `module.Module` 接口。
2. 为模块源文件添加对应的 `//go:build module_xxx` 标签，并在 `init()` 中调用 `module.Register`。
3. 在 `internal/imports/` 下添加带相同 build tag 的空白导入文件。
4. 将新 tag 加入 Makefile 的 `ALL_MODULES`。

## 平台说明

- 桌面 GUI 支持 Windows、Linux 和 macOS，具体构建依赖平台对应的 Fyne 图形库工具链。
- `module_jiggler` 的实际鼠标防离开功能仅支持 Windows。
- `module_banktool` 在 Windows 上可调用已安装的 Microsoft Excel；Go 转换器作为回退方案，并用于非 Windows 平台。
- MassUpload 模块需要运行时可访问项目提供的 Excel 模板文件。
