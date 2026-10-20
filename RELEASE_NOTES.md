# screen-cli 0.5.0

新增数据源凭证管理：credentials list、credentials preview --file -、credentials create --confirm。复用后台五种凭证类型、加密存储和域名隔离；预览不写入，确认绑定账号并在15分钟内生效，重复确认沿持久回执返回同一结果。

密钥仅通过文件或标准输入提交，不进入列表、预览或回执。Skill 包含完整字段和来源引用方式。需要同步更新服务端的 /cli/v1/credentials 接口；CLI 不会自动部署服务器，原页面上传检查和网页发布流程保留。

安装或升级：npm install --global @ghostedd/screen-cli@0.5.0。升级后执行 screen-cli skill install --agent codex（Claude Code 使用 --agent claude）。
