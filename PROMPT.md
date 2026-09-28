输入 4 KB 时看不出差别，64 KB 时差了三个数量级。rollhash 是 Go 1.24 的
重复子串索引库，只用标准库（无第三方依赖），构建走 go 工具链，
自检走 `scripts/check.sh`（`check/` 是固定验收程序，别改），既有用例走 `go test ./...`。

`rollhash.go` 里那份实现是对的，但每次查找都在做逐字节回溯，规模一大就不能用。

任务：把它改成不靠回溯的实现，同时守住 README 里的 8 条契约。判据是**计数**不是墙钟：
`bench.go` 里的 `Comparisons` 与 `HashOps` 两个计数器已给全，`check/` 会按规模卡上界。

验收：
- go test ./... 全绿，退出码 0；
- go run ./check 退出码 0，8 个场景全过（correctness 3 + cost 3 + collision 2）。

约束：
1. 不改 `check/`、不改 `bench.go` 里两个计数器字段的名字与含义；可以新增文件。
2. 对外 API 名与签名不许改；`rollhash_test.go` 里的既有用例一条都不许删或改。
3. 不引入第三方依赖，`go.mod` 里不许出现 `require`。
4. 哈希命中之后必须逐字节确认，结果要和朴素搜索完全一致。
