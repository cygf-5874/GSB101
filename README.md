# rollhash

文本上的重复子串索引库（Go 1.24，仅标准库，`go.mod` 无 `require`）。

`Index` 在一段文本上建索引，`Lookup` 找出一个模式出现的全部位置，`DupPairs` 找出相隔
固定距离的重复对。判据是**计数**而不是墙钟：`bench.go` 里给出两个计数器
（`Comparisons`、`HashOps`），固定件按规模卡上界。

```go
ix := rollhash.Index(text, 8)
ix.Lookup("mississippi")   // 全部起始下标，升序去重
ix.DupPairs(3)             // 所有 [i, i+3]，且 text[i:i+3] == text[i+3:i+6]
```

## 怎么跑

```bash
go build ./...             # 编译
go test ./...              # 既有用例（12 个）
go run ./check             # 验收场景（固定件，8 个）
go run ./check -list       # 列出全部场景
go run ./check --only cost
bash scripts/check.sh      # 同上，薄封装
go run ./repro             # 打印几个基准规模下的计数
```

前提：Go 1.24。不引入任何第三方依赖。

## 对外契约

`Idx` 的内部表示完全自由；下面 8 条是**对外语义**，**它们是契约，不是「当前行为」的转述**。

1. **公开 API 三件**：`Index(text string, k int) *Idx`、`(*Idx) Lookup(pat string) []int`、
   `(*Idx) DupPairs(k int) [][2]int`。`Lookup` 返回 `pat` 在文本里出现的全部起始下标；
   `DupPairs(k)` 返回所有相隔 `k` 的**重叠重复对** `[i, i+k]` —— 即满足 `i+2*k <= len(text)`
   且 `text[i:i+k] == text[i+k:i+2k]` 的那些 `i`（周期 `k` 的平方串起点），按下标升序。
   `Index` 的 `k` 是索引使用的窗口大小。
2. **`Lookup` 返回的匹配位置必须升序去重**；`k <= 0` 时 `Index` 返回**空索引**
   （其 `Lookup` 与 `DupPairs` 都返回空结果），`DupPairs` 的 `k <= 0` 时返回空切片。
3. **结果与调用次数无关**：同一个 `Idx` 上任意多次 `Lookup` / `DupPairs` 必须返回相同结果。
4. **计数口径**（计数在 `bench.go` 里定义）：`Comparisons` 统计「两字节是否相等」的比较次数；
   `HashOps` 统计「哈希窗口推进 / 计算」的次数（算一个窗口的哈希 = 1 次）。
5. **计数不得依赖墙钟或机器速度**，也不得为压低计数而**跳过正确性**：
   所有匹配位置必须与朴素实现对拍一致。
6. **语义等价要求**：对任意 `text`（长度 ≤ 4096）与任意 `pat`，`Lookup` 的结果集合与
   朴素暴力搜索**完全相同**；`DupPairs` 同理。
7. **哈希碰撞不得改变结果**：命中哈希后**必须逐字节复核**。本题统一的窗口哈希是
   `bench.go` 里的 `Hash`（多项式滚动哈希，基 257，对 `2^32` 取模）；索引里滚动推进出来的
   窗口哈希必须与它一致。命中哈希但字节不同的位置**绝不能**被报成匹配。
8. **边界**：空 `text`、空 `pat`、`pat` 长于 `text`、`text` 含重复长串、`k <= 0`，
   都不得 panic。

## 本次重构的目标

`rollhash.go` 里那份实现是对的（既有用例 12/12 全绿），但 `Lookup` 在每个位置都做逐字节回溯，
复杂度是 `O(len(text) * len(pat))`。

目标：把 `Lookup` 与 `DupPairs` 改成**不靠回溯**的实现，并在下面这个规模上同时满足
**正确性对拍**与**计数上界**：

- `len(text) == 65536`、`len(pat) == 64` 时，`Comparisons` 不得随 `len(text) * len(pat)` 增长，
  而应随 `len(text)` 近似线性：本机标定下目标 **< 2e5**（朴素实现约 4.2e6）；
- `HashOps` 必须随 `len(text)` 线性，且不为 0；
- 同一 `Idx` 上重复 `Lookup` 的计数必须逐次相同。

判据全部是硬编码或由固定件独立算出的期望值：不依赖墙钟、机器速度、随机源或 `map` 迭代顺序。

## 验收

`check/` 是固定验收程序，**不要修改**。场景分三组，共 8 个：

| 组 | 场景 | 覆盖 |
| --- | --- | --- |
| `correctness` | `CR1_random_vs_naive` | 契约 1、2、6：随机文本上 `Lookup` / `DupPairs` 与朴素对拍 |
| `correctness` | `CR2_repeated_long_vs_naive` | 契约 6、8：重复长串、全同字符上的对拍 |
| `correctness` | `CR3_edges_no_panic` | 契约 2、8：空文本 / 空模式 / 超长模式 / `k <= 0` 不 panic |
| `cost` | `CO1_comparisons_bound` | 契约 4、5：65536 规模的 `Comparisons` 上界 |
| `cost` | `CO2_hashops_bound` | 契约 4、5：65536 规模的 `HashOps` 上界（且 > 0） |
| `cost` | `CO3_calls_independent` | 契约 3、5：重复调用的结果与计数都相同且在上界内 |
| `collision` | `CL1_known_collision_matches_naive` | 契约 7：已知碰撞输入下结果仍与朴素一致 |
| `collision` | `CL2_byte_recheck_performed` | 契约 7：命中哈希后逐字节复核（反证哈希相同但字节不同的位置被挡住） |

`--only correctness|cost|collision` 可以只跑一组。每个场景的 panic 会被单独捕获并记成一条
`FAIL`，**一个场景挂掉不会遮蔽其余场景**。随机输入用固定种子的 LCG 生成，跨运行逐位可复现。

## 目录

```
.
├── .gitattributes
├── .gitignore
├── go.mod               module rollhash
├── bench.go             固定入口：Counts 计数器 + 统一的窗口哈希 Hash
├── rollhash.go          既有实现（朴素 O(n·m) 双指针，功能正确）
├── rollhash_test.go     既有用例（12 个）
├── PROMPT.md            本题的 User Prompt（逐字）
├── repro/main.go        打印基准规模下的计数
├── check/main.go        固定验收程序（勿改）
├── scripts/check.sh     薄封装：go run ./check "$@"
└── README.md
```

## User Prompt

```
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
```
