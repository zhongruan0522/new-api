# cmd/AGENTS.md

`cmd/` 存放可执行命令入口。当前唯一命令是 `server`，具体规则见 [server/AGENTS.md](server/AGENTS.md)。

- 命令入口保持薄封装，不承载业务逻辑或资源初始化。
- 每个命令目录必须有自己的 `AGENTS.md`，并写明验证命令。
