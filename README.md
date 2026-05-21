# staticrouter

`staticrouter` 当前定位为一套“静态配置发布系统”，而不是 Redis 路由数据库。

核心链路：

```text
XML / JSON
  -> RouteSnapshot
  -> Redis String
  -> Redis Stream
  -> Local Immutable Route Table
  -> Atomic Swap
  -> Runtime Lookup
```

## 设计原则

- Redis 只存固定格式的整表快照
- 运行时查询完全走本地内存
- 内存里存的是编译后的高性能关系映射，不是原始快照数据
- 路由更新只做整表替换，不做单条 CRUD
- 整表替换由一个总版本号控制
- 快照带 `checksum`
- 配置源支持 XML / JSON
- `RouteRecord` 不携带地址，地址由其他服务发现系统提供

## 数据模型

### RouteContext

查询条件：

```proto
message RouteContext {
  string kind = 1;
  string node_type = 2;
  int32 route_key = 3;
}
```

### RouteRecord

一条静态路由原始记录：

```proto
message RouteRecord {
  string kind = 1;
  string node_type = 2;
  repeated int32 route_keys = 3;
  int32 route_key_start = 4;
  int32 route_key_end = 5;
  string node_id = 6;
  map<string, string> metadata = 7;
}
```

支持两种模式：

- 离散键：
  `route_keys=[1001,1002,1003]`
- 连续区间：
  `route_key_start=2000`
  `route_key_end=2099`

注意：

- `RouteRecord` 不再携带 `address`
- 地址由其他服务发现系统根据 `node_id` 和运行时状态提供

### RouteSnapshot

整表快照：

```proto
message RouteSnapshot {
  int64 version = 1;
  string scope = 2;
  string checksum = 3;
  repeated RouteRecord routes = 4;
}
```

其中：

- `version` 是整表大版本号
- `scope` 是路由归属类别，例如 `dev` / `qa` / `review` / `prod`
- `checksum` 是快照内容摘要

## 配置源结构

当前支持：

- XML
- JSON

两者结构保持镜像一致，后续如果继续加 YAML / TOML，也建议复用同一套层级语义。

### XML 示例

```xml
<routes version="7" scope="qa">
  <route>
    <kinds>
      <kind>player</kind>
      <kind>mail</kind>
    </kinds>
    <nodes>
      <node node_id="game-node-1">
        <route_keys>
          <keys>
            <key>1001</key>
            <key>1002</key>
          </keys>
          <ranges>
            <range start="2000" end="2099" />
          </ranges>
        </route_keys>
      </node>
    </nodes>
  </route>
</routes>
```

### JSON 示例

```json
{
  "version": 8,
  "scope": "dev",
  "routes": [
    {
      "kinds": {
        "kind": ["player", "mail"]
      },
      "nodes": {
        "node": [
          {
            "node_id": "game-node-1",
            "route_keys": {
              "keys": {
                "key": [1001, 1002]
              },
              "ranges": {
                "range": [
                  { "start": 2000, "end": 2099 }
                ]
              }
            }
          }
        ]
      }
    }
  ]
}
```

### 结构解释

- 一个 `route` 表示一组共享规则
- `kinds` 描述这组业务服务类别
- `nodes` 描述承载这些类别的节点
- 每个 `node` 定义自己的 `keys` 和 `ranges`
- 解析时会展开成：
  `每个 kind x 每个 node = 一条 RouteRecord`

## 加载入口

当前提供：

- `LoadRouteSnapshotFromFile(path)`
- `Router.ReplaceAllFromFile(ctx, path)`
- `Publisher.PublishFile(ctx, path)`

其中：

- `LoadRouteSnapshotFromFile` 会按扩展名自动选择解析器
- 当前支持：
  - `.xml`
  - `.json`

## Scope 规则

`RouteSnapshot.scope` 表示这套路由归属哪个环境/类别。

例如：

- `dev`
- `qa`
- `review`
- `prod`

节点启动时会带自己的 `scope`，只会：

- `GetSnapshot(ctx, scope)`
- `Watch(ctx, scope)`

也就是说：

- `qa` 节点只看 `qa` 的 routers
- `prod` 节点只看 `prod` 的 routers
- 不同归属的静态路由快照在 Redis 中相互隔离

## 发布流程

推荐的发布流程：

```text
1. 维护 XML / JSON 原始配置
2. 读取配置并生成 RouteSnapshot
3. 规范化快照
   - 清空旧 checksum
   - 重新计算 checksum
4. 编译并校验快照
   - duplicate exact key
   - exact/range overlap
   - range/range overlap
5. 发布到 Redis
   - SET staticrouter:snapshot:{scope}
   - XADD staticrouter:events:{scope}
6. 各节点按自己的 `scope` watch stream
7. 拉取/接收新 snapshot
8. 本地编译成 immutable route table
9. atomic swap
```

当前仓库里推荐的发布入口是：

- `NewPublisher(store)`
- `Publisher.PublishFile(ctx, path)`

对业务进程来说，推荐只使用更小的运行时入口：

- `Init(opts ...Option)`
- `UpdateConfig(mode, content)`
- `GetRoute(routeCtx)`

其中：

- `Init` 只负责初始化运行时、读取 Redis 中的当前快照、启动 watch、构建本地表
- `UpdateConfig` 负责外部主动推送新的 XML / JSON 配置内容
- `GetRoute` 只查本地内存路由表

示例：

```go
err := staticrouter.Init(
    staticrouter.WithScope("qa"),
    staticrouter.WithRedisConfig(staticrouter.RedisConfig{
        Host:      "192.168.0.138:7000",
        Password:  "123456",
        IsCluster: true,
    }),
)
```

外部系统同步新配置：

```go
err := staticrouter.UpdateConfig(staticrouter.ConfigModeXML, xmlContent)
```

运行时查询：

```go
route, ok := staticrouter.GetRoute(&staticrouter.RouteContext{
    Kind:     "player",
    NodeType: "game",
    RouteKey: 1001,
})
```

`GetRoute` 返回的是本地 immutable route table 中的记录指针，调用方应按只读数据使用，不要修改返回的 `RouteRecord`。

共享 protobuf 类型实际位于 `server/staticrouter/model` 包，主包保留了类型别名，方便业务侧继续使用 `staticrouter.RouteContext` / `staticrouter.RouteRecord`。

## Redis 存储模型

Redis 当前只存两个东西。

### 1. 整表快照

```text
staticrouter:snapshot:{scope}
```

值为 protobuf 编码的 `RouteSnapshot`。

### 2. 快照变更事件

```text
staticrouter:events:{scope}
```

值为 protobuf 编码的 `RouteSnapshot`，通过 Redis Stream 发布。

也就是说：

- Redis 不负责 route lookup
- Redis 不维护 exact index / range index
- Redis 不是实时路由数据库
- Redis 只是配置分发器
- 不同 `scope` 的快照和事件流分别存放

## Router 运行时模型

`Router` 启动后会把 `RouteSnapshot` 编译成一份本地不可变路由表：

- exact map
- 按 `kind + node_type` 分组并排序的 range slice

查询逻辑：

1. 先查 exact
2. 再查 range
   - 使用排序后二分查找

这份表不做原地修改，而是：

```text
build new table
  -> atomic swap
```

当前实现已切到原子切换：

- `atomic.Pointer[compiledSnapshot]`

## Reload 流程

节点启动：

```text
GetSnapshot(scope)
  -> NormalizeSnapshot
  -> compileSnapshot
  -> atomic swap
  -> Watch(scope)
```

收到新快照：

```text
NormalizeSnapshot
  -> compileSnapshot
  -> version compare
  -> atomic swap
```

如果 watch 通道断开：

```text
reconnect watch
  -> GetSnapshot(scope) again
  -> compile latest snapshot
  -> atomic swap
```

这保证：

- watch 中断后不会只“继续听”
- 会主动重新拉最新整表
- stream 天然支持重放

## 版本与 checksum 规则

当前实现约束：

- 新快照会自动重新计算 `checksum`
- watch 收到旧版本快照不会覆盖当前版本
- store 侧不允许版本回退写入

也就是说：

- `version` 用来控制全局先后顺序
- `checksum` 用来识别内容一致性

## 运维命令示例

### 1. 本地校验

```bash
go test ./server/staticrouter/... -v
```

### 2. 发布配置快照

推荐的命令形态：

```bash
go run ./cmd/staticrouter-publisher \
  -config ./configs/staticrouter/routes.xml \
  -redis 192.168.0.138:7000 \
  -password 123456 \
  -cluster
```

或者：

```bash
go run ./cmd/staticrouter-publisher \
  -config ./configs/staticrouter/routes.json \
  -redis 192.168.0.138:7000 \
  -password 123456 \
  -cluster
```

### 3. 真实 Redis BDD 验证

```bash
STATICROUTER_RUN_INTEGRATION=1 \
STATICROUTER_REDIS_HOST=192.168.0.138:7000 \
STATICROUTER_REDIS_PASSWORD=123456 \
go test ./server/staticrouter -run TestBDD -v
```

### 4. 全量回归测试

```bash
go test ./... -v
```
