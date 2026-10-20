# screen-cli 0.4.0

新增 npm 独立安装方式：`npm install --global screen-cli@0.4.0`。

- npm 包直接包含 macOS、Windows、Linux 的 x64/ARM64 二进制，安装不执行脚本，不额外从 GitHub 下载运行文件。
- 启动时核对发布版本与当前平台二进制哈希，保留原 CLI 配置、浏览器授权与受限权限。
- 使用 `screen-cli skill install --agent codex` 或 `--agent claude` 安装随包 Skill；更新保留其他 Skill 与用户修改。
- 保留直接下载二进制、Skill ZIP 与 SHA256SUMS 的方式。发布仍由网页确认，CLI 不具有正式发布权限。
