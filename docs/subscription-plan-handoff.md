# 订阅套餐功能交接文档

## 1. 当前状态

- 跟踪清单：`docs/subscription-plan-tracking.csv`
- 总项数：24
- 已完成：24
- 阻塞：0
- 跳过：0
- 未完成：0
- CSV 中 `是否完成`、`是否commit`、`是否通过review` 已全部回填为“是”
- 最新一轮自检已通过：`go test ./...`、`bun run build`
- 最新一轮复核未再返回新的 P0/P1 阻塞项

## 2. 已确认的核心业务规则

- 用户有有效套餐时，API 调用优先扣套餐额度，不回退余额。
- 套餐用户只能使用 `support_subscription` 的渠道；没有可用渠道时直接报错。
- 单用户同一时刻只允许一个活跃套餐。
- 套餐购买、续订、升级使用独立订阅订单，不复用充值订单。
- 升级规则为“立即补差升级”。
- 升级后重开完整周期，旧套餐保留总已用额度。
- 刷新周期按订阅生效时间滚动。
- 实际消耗超过预扣且额度不足时，允许末次超额并记录。

## 3. 已落地范围

### 3.1 后端数据与服务

已新增套餐核心模型并接入迁移：

- `model/subscription.go`
  - `SubscriptionPlan`
  - `UserSubscription`
  - `SubscriptionOrder`
  - `SubscriptionBill`
  - `SubscriptionRedemption`
- `model/main.go`

已新增套餐服务主链路：

- `service/subscription.go`
  - 购买
  - 续订
  - 升级
  - 管理员发放/移除
  - 兑换码兑换
  - 订阅订单完成

已接入套餐设置：

- `setting/operation_setting/subscription_setting.go`

### 3.2 支付与订单履约

已把套餐订单从充值订单中拆开处理：

- `controller/topup.go`
- `controller/topup_stripe.go`

当前状态：

- 现金支付回调可区分充值订单和套餐订单。
- Stripe 回调可区分充值订单和套餐订单。
- 不会再把套餐付款误记为余额充值。
- 已阻止重复创建未完成的现金套餐订单。

### 3.3 API 扣费与渠道筛选

已接入套餐计费资金源与渠道筛选：

- `service/funding_source.go`
- `service/billing_session.go`
- `middleware/auth.go`
- `middleware/distributor.go`
- `service/channel_select.go`
- `model/channel_cache.go`
- `model/ability.go`
- `relay/helper/price.go`
- `model/channel.go`

当前状态：

- 有效套餐用户优先走套餐额度。
- 套餐用户只能使用支持套餐的渠道。
- 高优先级渠道不支持套餐时，可以继续尝试低优先级套餐渠道。
- SQLite 下 `FOR UPDATE` 不兼容点已修正为仅非 SQLite 使用。

### 3.4 接口与鉴权

已新增订阅相关接口并接入权限：

- `controller/subscription.go`
- `router/api-router.go`

用户接口需登录，管理接口需管理员权限。

### 3.5 前端页面与入口

已新增或接入以下页面/入口：

- `web/src/pages/Subscription/index.jsx`
- `web/src/pages/SubscriptionPlan/index.jsx`
- `web/src/pages/SubscriptionRedemption/index.jsx`
- `web/src/components/settings/SubscriptionSetting.jsx`
- `web/src/App.jsx`
- `web/src/components/layout/SiderBar.jsx`
- `web/src/hooks/common/useSidebar.js`
- `web/src/pages/Setting/Operation/SettingsSidebarModulesAdmin.jsx`
- `web/src/components/table/channels/modals/EditChannelModal.jsx`
- `web/src/components/table/users/modals/EditUserModal.jsx`
- `web/src/pages/DynamicRatio/index.jsx`
- `web/src/helpers/render.jsx`

当前状态：

- 系统设置已支持套餐设置。
- 个人中心已支持订阅管理入口。
- 管理端已支持套餐管理、套餐兑换码菜单。
- 渠道编辑已支持“是否支持套餐用户调用”。
- 用户编辑弹窗已支持套餐发放、移除、追加下周期套餐。
- 动态倍率规则已支持“是否对套餐生效”。

### 3.6 模型广场

已完成基础接入：

- `model/pricing.go`
- `web/src/components/table/model-pricing/filter/PricingQuotaTypes.jsx`
- `web/src/hooks/model-pricing/useModelPricingData.jsx`
- `web/src/hooks/model-pricing/usePricingFilterCounts.js`
- `web/src/components/table/model-pricing/modal/components/ModelPricingTable.jsx`

当前状态：

- 已支持“套餐计费”筛选。
- 已支持展示套餐支持标记、套餐倍率、当前套餐动态倍率、实际倍率。
- 已支持在模型详情中按分组展示套餐倍率后的提示/补全/额外计费/按次价格；分段计费模型按上下文区间展示倍率后价格。
- 套餐设置页已从手写 JSON 改为跟随支持套餐渠道模型动态加载的倍率表。
- 支持套餐渠道、渠道能力、套餐倍率设置变更后会刷新定价缓存，避免等待默认 1 分钟缓存过期。

## 4. 未完成项

无。

## 5. 已验证内容

已执行：

```bash
/usr/bin/env GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./...
cd web && bun run build
```

说明：第一次在默认沙箱内执行 `go test ./...` 时，已有 `httptest` 用例因无法监听本地端口报 `socket: operation not permitted`；使用同一命令并允许本地测试监听后通过。

## 6. 交接建议

无剩余实现项。后续变更建议继续以 `docs/subscription-plan-tracking.csv` 为验收清单，并在改动支付回调或订阅订单履约时重点复测现金支付和 Stripe 回调分发。

## 7. 关键文件索引

- 跟踪清单：`docs/subscription-plan-tracking.csv`
- 本交接文档：`docs/subscription-plan-handoff.md`
- 订阅模型：`model/subscription.go`
- 订阅服务：`service/subscription.go`
- 套餐设置：`setting/operation_setting/subscription_setting.go`
- 套餐接口：`controller/subscription.go`
- 路由注册：`router/api-router.go`
- 套餐扣费：`service/funding_source.go`
- 计费会话：`service/billing_session.go`
- 渠道筛选：`middleware/distributor.go`、`service/channel_select.go`
- 套餐价格计算：`relay/helper/price.go`
- 用户页：`web/src/pages/Subscription/index.jsx`
- 套餐管理页：`web/src/pages/SubscriptionPlan/index.jsx`
- 套餐兑换码页：`web/src/pages/SubscriptionRedemption/index.jsx`
- 套餐设置页：`web/src/components/settings/SubscriptionSetting.jsx`
- 模型广场详情：`web/src/components/table/model-pricing/modal/components/ModelPricingTable.jsx`
