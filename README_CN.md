# Privatize

一个将 Go 第三方依赖源码嵌入项目并重写 import 路径的命令行工具。

## 背景

第三方库有时存在缺陷或功能缺失，严重影响业务开发。vendor 模式不允许修改代码。Privatize 通过将依赖变为项目的一部分来解决这个问题，你可以自由地修补它。

## 工作原理

1. 读取 `.privatize.yaml` 配置
2. 创建临时 Go 模块，执行 `go mod vendor` 获取源码
3. 通过 AST 解析重写 vendor 源码中的 import 路径
4. 将重写后的源码复制到项目中
5. 重写项目 `.go` 文件中的 import 路径

## 安装

```bash
go install github.com/edoger/privatize/cmd/privatize@latest
```

## 使用

### 初始化配置

```bash
cd your-project
privatize init
```

生成 `.privatize.yaml`：

```yaml
imports: []

rules: {}

exclude:
  - golang.org/x
```

### 配置

编辑 `.privatize.yaml`，指定需要私有化的包：

```yaml
imports:
  - github.com/user/pkg
  - github.com/other/lib

rules:
  github.com/user/pkg: internal/pkg
  github.com/other/lib: vendor/lib

exclude:
  - golang.org/x
  - my.org/internal
```

**字段说明：**
- `imports`: 需要私有化的包列表，用于生成临时 Go 模块依赖
- `rules`: 原始导入路径到项目内相对路径的映射。子包路径自动推导
- `exclude`: 前缀匹配的排除列表，匹配的包不会被私有化

**带版本号的包**（如 `/v2`）必须显式声明：

```yaml
rules:
  github.com/user/pkg: internal/pkg
  github.com/user/pkg/v2: internal/pkgv2
```

### 预览变更

```bash
privatize run --dry-run
```

展示将要修改的内容，不实际修改文件。

### 执行

```bash
privatize run
```

执行私有化流水线。

## 示例

项目模块为 `github.com/foo/bar`，配置如下：

```yaml
imports:
  - github.com/user/pkg

rules:
  github.com/user/pkg: internal/pkg

exclude:
  - golang.org/x
```

处理前：

```go
import "github.com/user/pkg"
import "github.com/user/pkg/sub"
```

处理后：

```go
import "github.com/foo/bar/internal/pkg"
import "github.com/foo/bar/internal/pkg/sub"
```

`github.com/user/pkg` 的源码被复制到 `internal/pkg/`，所有 import 已重写。

## 路径映射规则

| 原始路径 | 规则 | 结果 |
|---|---|---|
| `github.com/user/pkg` | `-> internal/pkg` | `github.com/foo/bar/internal/pkg` |
| `github.com/user/pkg/sub` | 父包规则自动推导 | `github.com/foo/bar/internal/pkg/sub` |
| `github.com/user/pkg/v2` | 无显式规则 | **不处理** |
| `golang.org/x/text` | 被排除 | **不处理** |

## 技术细节

- 使用 `go/parser` 的 `parser.ImportsOnly` 模式进行 AST 级别的 import 检测
- 字节级替换，保留 import 之外的文件结构不变
- 支持所有导入形式：命名、空白导入（`_`）、点导入（`.`）、别名导入
- 临时工作区位于系统临时目录，执行完毕自动清理
- 目录级完整复制，包含所有非 Go 文件（数据文件、模板等）

## 依赖

- [cobra](https://github.com/spf13/cobra) - CLI 框架
- [bubbletea](https://github.com/charmbracelet/bubbletea) - 终端 UI
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML 配置解析
- [x/mod](https://golang.org/x/mod) - go.mod 解析

## 许可证

MIT
