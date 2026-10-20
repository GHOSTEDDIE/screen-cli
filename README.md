# screen-cli

独立的大屏开发客户端。用户的本机 Agent 写源码，系统负责工程准入、检查、预览和网页发布确认。客户端不读取数据库、不执行服务器 shell，也不拥有发布权限。

使用前需同步更新系统后端、前端及沙箱控制器；旧版本没有 CLI 接口。后端沿原启动迁移创建上传回执表，继续使用既有 Redis、沙箱与模型设置。本次交付包不自动部署服务端。

## npm 安装（推荐）

需要 Node.js 18 或以上。npm 包直接携带六平台二进制与完整 Skill；无需 Go 编译器，不在安装时执行脚本或从 GitHub 下载运行文件。

```sh
npm install --global screen-cli@0.4.0
screen-cli --version
screen-cli skill install --agent codex
# Claude Code 使用：screen-cli skill install --agent claude
screen-cli config set --endpoint https://系统地址 --web-url https://网页地址
screen-cli auth login --account 用户名
```

Skill 安装使用当前用户的 Agent 目录，支持 CODEX_HOME 与 CLAUDE_CONFIG_DIR。重复安装相同内容不改文件；官方版本可原子更新，已有自定义内容或其他 Skill 会保留并要求先核对。npm 包的本地 Skill 安装命令不扩大系统账号权限。需要一次性调用时可使用 `npx --yes --package=screen-cli@0.4.0 screen-cli --version`。

独立分发仓库：[GHOSTEDDIE/screen-cli](https://github.com/GHOSTEDDIE/screen-cli)。此仓库只包含客户端、Skill 与发布脚本，没有后台实现或私有配置。

## 二进制安装与登录

交付目录包含 macOS、Linux、Windows 的 amd64/arm64 二进制和 SHA256SUMS。安装到自己的命令目录，不需要 Go 或 Node 编译器：

```sh
sh scripts/install.sh --from dist --bin-dir "$HOME/.local/bin"
screen-cli config set --endpoint https://系统地址 --web-url https://网页地址
screen-cli config show
screen-cli auth login --account 用户名
```

Windows 使用 `scripts/install.ps1 -From dist -BinDir "$env:LOCALAPPDATA\screen-cli\bin"`。系统与网页同站时省略 `--web-url`。本机开发可使用 `http://127.0.0.1:<端口>`；远程地址要求 HTTPS。无自动浏览器环境使用 `auth login --no-browser`，按输出的地址在同一电脑的浏览器授权。

登录配置默认保存到操作系统用户配置目录，可通过 `SCREEN_CLI_CONFIG_DIR` 指定 CLI 专用配置目录。配置目录与凭据文件使用仅本人访问权限；更换系统地址会清除之前的身份。访问凭据有效期 15 分钟，刷新凭据轮换；浏览器登录退出或过期后 CLI 授权同时失效。

## 后台下载与 Agent Skill

后台「智能搭建 → 本机开发」按操作系统、处理器下载 CLI、Agent Skill ZIP 和校验和。当前版本随生产镜像构建六平台二进制；服务端不在请求期间编译。开发环境先运行 `sh cli/scripts/build.sh`，后端默认读取 `cli/dist`；也可用 `SCREEN_CLI_RELEASE_DIR` 指定发布目录。目录必须包含同一版本完整的 manifest、二进制、Skill 和校验和，缺失或变更时拒绝下载。下载接口要求正式网页登录，受限 CLI 令牌不能代替后台会话。

Skill ZIP 内的 `screen-cli` 文件夹放入 Coding Agent 的技能目录，Codex 为 `~/.codex/skills/`，Claude Code 为 `~/.claude/skills/`。它指导 Agent 由本机编写源码，通过 CLI 上传检查，不默认调用内置生成 Agent，也不直接改数据库或绕过网页发布。

`config set` 原子保存服务端和网页地址；`--endpoint` 与兼容参数 `--server` 同义，冲突时拒绝。`config show` 只返回地址、账号和凭据存在状态，不输出令牌。`auth login --account 用户名` 校验浏览器账号；错误账号不能消耗授权请求。`auth status` 从服务端读取实际身份；`auth logout` 撤销受限授权，已失效的授权也可清理本机凭据。网络不可达时暴露错误，避免误报远端已撤销。

## 开发流程

新增数据源可以全程通过 CLI 完成：

```sh
screen-cli sources test --file connection.json
# 从测试结果选择表编号，写入 connection.json 的 allowed_tables
screen-cli sources preview --file connection.json
screen-cli sources create --confirm <接入预览凭据>
```

例如飞书多维表格连接文件为 `{"name":"机加工示例","kind":"feishu_bitable","config":{"app_name":"新多维表格标识"},"allowed_tables":["测试结果中的表编号"]}`。使用系统已配置的数据凭证；指定其他已配置凭证时，在 config 中提供 `certificate_id`。数据库连接同网页格式，密码通过 `--file -` 从标准输入提交，不保存到页面工程。CLI 不输出连接密码、应用密钥或原始连接配置。

`test` 和 `preview` 不写入数据来源。预览只显示选定表及范围，有效期15分钟，确认时重新检查连接、凭据、规范及任务范围。创建和回执在同一事务保存；重复确认返回同一来源，不重复创建。可添加 `--task-id <已有任务编号>`，创建同时授权给本人仍有效的任务；不指定时只创建来源，随后用返回的来源编号初始化页面。

```sh
screen-cli sources
screen-cli init ./运营大屏 --name "运营大屏" --source-ids 1,2
cd ./运营大屏
screen-cli context
# 让本机 Agent 读取 AGENTS.md 和 .screen/LOCAL_AGENT.md 并修改源码
screen-cli push --preview
screen-cli push --confirm <预检结果的确认凭据>
screen-cli check --wait
screen-cli preview
```

拉取已有页面：`screen-cli pull --page-id 12 --source-ids 1,2`。本地文件将被覆盖时先返回差异，添加 `--confirm` 才执行。任意命令可以添加 `--project <目录>`。

`init` 会在系统保存新页面草稿及本机开发任务，复用账号页面创建权限；不只是生成本地目录。CLI 没有 `publish` 命令，其凭据也不能调用后台发布接口。`check` 检查通过后，由 `preview` 打开原工作台，用户确认发布该不可变版本。

0.2.1 修复预览地址：`preview` 返回并打开 `/setting?task_id=任务编号`，直接进入后台“大屏搭建”并选择当前任务。它不再生成旧版 `/ai-agent` 助手链接。服务端 `context`、`status` 的 `web_path` 使用同一地址；旧任务链接只作重定向兼容。

系统搭建规范变更后，已有本地工程先执行 `screen-cli rules sync` 更新同源规范与锁定依赖清单。使用 `screen-cli plan get` 回读设计计划；绑定指标后将计划保存为 `.screen/design-plan.json`，执行 `screen-cli plan save --file .screen/design-plan.json --confirm`。完整格式见 `.screen/LOCAL_AGENT.md`。源码仪表使用对应的 `data-screen-section` 编号；正式检查核对计划、真实绑定与各视口。计划变化会让之前的上传预检与检查结果失效，需要重新预检、检查。

上传 ZIP 时在预检和确认两次命令添加相同的 `--zip <路径>`。包必须符合系统模板；不能上传普通网站、后端应用、安装脚本、依赖目录或编译产物。目录打包明确排除 `.git`、`.screen`、`.agents`、`.codex`、node_modules、dist、out、build 和 __MACOSX，并返回排除清单。其他隐藏文件、链接直接拒绝。ZIP 中违规条目不静默删除。

服务端执行单文件 2 MiB、源码合计与 ZIP 20 MiB、256 个源码文件限制；元数据只接受系统原文。依赖、数据权限、真实数据和视觉检查继续使用系统规则。图片通过 assets 命令独立上传，源码只引用素材编号。

完整的本机 Agent 操作指南由系统随模板提供至 `.screen/LOCAL_AGENT.md`，包括数据样例、变更申请、素材、错误修正和确认步骤。数据接入、指标保存、绑定和依赖申请在原工作台的确认卡片处理；正式发布由用户在网页预览后确认。

## 自动化与构建

默认输出 JSON。退出码：0 成功、1 输入/服务/检查错误、2 需要登录、3 草稿或预检冲突。检查失败通过 `status` 返回保存的日志与视觉报告；预检 `saved:false` 不代表草稿已保存。

```sh
go test -race ./...
sh scripts/build.sh --version 0.2.1
```

该目录是独立 Go 模块，标准库即可构建。release 工作流只生成可下载的交付物；不会自动部署系统或发布用户页面。

Windows 配置目录使用当前用户的访问控制列表，规则依据 [Microsoft icacls 文档](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/icacls)。

## 独立仓库同步与发布

唯一源码来源为系统仓库的 cli/。使用 `node scripts/export-repository.mjs 目标目录` 导出白名单文件；已有镜像出现非白名单内容或本地改动会拒绝覆盖。发布前执行 `node scripts/export-repository.mjs --check 目标目录`。

先同步 VERSION 与 package.json，再执行 Go/npm 测试、`sh scripts/build.sh` 和 `npm pack --dry-run`。独立仓库的 release.yml 在三个操作系统运行测试并构建发布文件，推送对应 v版本 标签后发布 GitHub Release。初次 npm 发布使用账号的正常发布权限和 npm 所需确认；后续可按 npm 官方流程将此仓库的 npm-publish.yml 登记为可信发布者，在版本标签上手动运行。工作流不保存 npm 长期令牌。
