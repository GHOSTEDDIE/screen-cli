# 命令与安装

以安装版本的 `screen-cli --help` 与工程 `.screen/LOCAL_AGENT.md` 为准。所有命令输出稳定 JSON：成功 `ok:true,data`，失败 `ok:false,error`；登录等待会先输出授权地址。退出码 0 成功、1 一般失败、2 认证、3 冲突。

## 安装

优先执行 `npm install --global screen-cli@0.4.0`，随后执行 `screen-cli skill install --agent codex`（Claude Code 改为 `--agent claude`）。npm 包自带二进制和 Skill，不需要 Go 或 GitHub 运行时下载。已有登录配置沿用，安装后执行 `screen-cli --version` 和 `screen-cli auth status` 核对。

无 npm 时，后台按操作系统和处理器下载 CLI、SHA256SUMS。macOS/Linux 用 `shasum -a 256 文件名` 或 `sha256sum 文件名` 对照校验，再将文件改名为 `screen-cli` 并 `chmod +x`，放入用户 PATH。Windows 用 `Get-FileHash 文件名 -Algorithm SHA256` 校验，改名为 `screen-cli.exe` 并加入用户 PATH。不需 Go、Node 或系统管理员权限。

Skill ZIP 解压后将整个 `screen-cli` 文件夹放入 Agent 的 Skill 目录；Codex 通常为 `~/.codex/skills/`，Claude Code 通常为 `~/.claude/skills/`。保留其中 `SKILL.md`、agents 和 references。不要把发布 ZIP 当作大屏源码上传。

```sh
screen-cli config set --endpoint https://系统地址 --web-url https://网页地址
screen-cli config show
screen-cli auth login --account 用户名
screen-cli auth status
screen-cli auth logout
screen-cli sources list
screen-cli init ./大屏 --kind react --name "生产运营" --source-ids 1,2
screen-cli --project ./大屏 pull --page-id 123
screen-cli --project ./大屏 rules sync
screen-cli --project ./大屏 context
screen-cli --project ./大屏 data --operation 操作 --file 参数.json
screen-cli --project ./大屏 plan get
screen-cli --project ./大屏 plan save --file .screen/design-plan.json --confirm
screen-cli --project ./大屏 propose --file 申请.json
screen-cli --project ./大屏 assets list
screen-cli --project ./大屏 assets add --file 图片.png --confirm
screen-cli --project ./大屏 push --preview
screen-cli --project ./大屏 push --confirm 预检凭据
screen-cli --project ./大屏 check --wait
screen-cli --project ./大屏 status
screen-cli --project ./大屏 preview --no-open
screen-cli --project ./大屏 cancel
```

ZIP 上传在预检与确认时都加相同 `--zip 文件.zip`。源码单文件 2 MiB、工程与 ZIP 20 MiB、256 文件；图片单张 10 MiB、版本合计 64 MiB。限制由服务端最终执行，不在 Skill 维护另一套策略。

## 新增数据来源

先核对用户授权、来源类型和表范围，复用已有凭证。具体连接结构按原网页接入配置或服务端模板，不猜测字段：

```sh
screen-cli sources test --file 连接.json
screen-cli sources preview --file 连接.json --task-id 当前任务编号
screen-cli sources create --confirm 接入预览凭据
```

`test` 和 `preview` 不创建来源。使用测试返回的表编号填写 `allowed_tables`，只在用户确认预览范围后执行 `create`。密码通过 `--file -` 标准输入提交，不写入页面工程；CLI 不应输出密钥。第三方数据服务授权与系统来源创建是不同步骤，不能以一方成功替代另一方验收。
