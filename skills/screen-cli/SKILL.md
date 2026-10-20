---
name: screen-cli
description: 使用 screen-cli 在本机创建或修改大屏、读取授权数据、上传源码和检查预览。适用于 Codex、Claude Code 等 Coding Agent 向大屏系统交付产物，不用于普通全栈应用或直接发布。
---

# 本机大屏开发

由你在本机编写源码，通过独立 `screen-cli` 上传产物。不要默认调用系统生成 Agent，不修改系统数据库或服务器源码来交付页面。

## 连接与身份

1. 运行 `screen-cli --version`。需要安装时从系统后台「智能搭建 → 本机开发」下载当前平台的 CLI、SHA256SUMS，校验后安装至用户命令目录。Skill 与 CLI 使用同一发布版本。
2. 用用户提供的地址执行 `screen-cli config set --endpoint https://系统地址`。网页与接口地址不同才加 `--web-url https://网页地址`；不要猜测地址。
3. 执行 `screen-cli config show`、`screen-cli auth status`。未登录时执行 `screen-cli auth login --account 用户名`，让用户在浏览器确认真实账号。无自动浏览器时加 `--no-browser`，使用返回的授权链接；不要读取浏览器 Cookie、测试账号文件或尝试绕过授权。
4. 配置在操作系统用户目录，禁止复制凭据到项目、Skill、对话或提交。更换服务端会清除旧身份。401 需要重新授权，403 核对权限，不能换用其他账号。

## 开发与交付

- 新建：`screen-cli init ./大屏 --kind react --name "运营大屏"`；会真实创建草稿和本机开发任务。先核对用户是否要求新建。HTML 使用 `--kind html`。
- 修改：在独立目录执行 `screen-cli --project ./大屏 pull --page-id 页面编号`。覆盖提示出现时先展示差异，取得具体覆盖授权后再加 `--confirm`。
- 读取工程 `AGENTS.md`、`.screen/LOCAL_AGENT.md` 和 `screen-cli --project ./大屏 context`。这些是服务端同源规范与任务数据范围。已有工程规范过期时用 `rules sync`，不要手动改锁定依赖或规范元数据。
- 使用授权真实数据和 `screenSDK`；先读取目录和字段，最多 20 行脱敏样例不能代表全量业务结果。新增数据源、指标绑定、依赖遵循原确认卡片；详细命令按 [commands.md](references/commands.md) 和模板指南。
- 开发单页面 React 或 HTML 大屏，沿系统入口与锁定依赖。图片使用 `assets add` 单独上传。不要打包 node_modules、构建产物、密钥、服务端、安装/构建脚本或自定义运行环境；不要用截图替代可运行页面。
- 保存设计计划后运行 `screen-cli --project ./大屏 push --preview`，向用户展示真实新增、修改、删除和排除项。获得对该差异的确认后，运行 `push --confirm 预检凭据`；15 分钟过期或文件、权限、策略、草稿变化必须重新预检。
- 执行 `screen-cli --project ./大屏 check --wait`，根据真实诊断在本机修正，再预检、确认上传、检查。不要要求系统生成 Agent 自动覆盖本机产物。取消使用 `cancel`；停止后不重启旧检查或提交。
- 执行 `screen-cli --project ./大屏 preview --no-open`，回读 `status`，向用户给出返回的工作台地址和页面编号。正式发布仍由用户在网页预览并确认，CLI 没有发布权限。

报告源码变更、CLI 回执、检查版本与诊断、实际页面回读和发布状态。编译成功或上传成功不能代表真实数据、视觉检查或已发布。错误不反复盲重试：身份问题停止待登录，冲突重新拉取比较，检查问题修正对应原因。
