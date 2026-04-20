# CLAUDE_TODO.md — DC-Manager 大版本升级任务清单

> **项目**: 数据中心IT设备管理系统 (dc-manager)
> **技术栈**: Go (Gin + GORM + SQLite) + React (Vite + Ant Design + ECharts)
> **部署**: Docker Compose (前后端分离)
> **生成日期**: 2026-04-12
> **需求来源**: 需求.txt + 001_中联重科数据中心台账_20260226.xlsx

---

## 项目现状摘要

### 已有功能
- ✅ 设备台账 CRUD（品牌/型号/序列号/IP/机房/机柜/U位等基础字段）
- ✅ 设备导入导出（Excel）
- ✅ 巡检记录管理（含图片上传）
- ✅ 巡检大屏 Dashboard（ECharts 统计图表）
- ✅ 用户管理 + 角色管理（RBAC 权限）
- ✅ JWT 认证 + 密码管理
- ✅ MCP 协议接入
- ✅ Docker Compose 部署

### 已有数据模型（backend/models/）
- `Device`: 32 个字段（含 source/status/datacenter/cabinet/u_position 等，状态为简单字符串）
- `Inspection`: 巡检记录（关联 Device，含 severity/status）
- `User` / `Role`: 用户角色（权限 JSON 数组存储）

### 现有数据特征（Excel 台账分析）
- 18 个 Sheet，涵盖：存储、数据中心、总部大楼、高机、土方、泵送、科技园、移动、异地园区等
- 3 个机房落位图 Sheet（1-1机房/1-2机房/2-1机房），每个 Sheet 用列表示机柜、行表示 U 位，记录设备落位信息
- 机柜高度统一 47U，机柜编号格式如 A-01、C-01、HDA A-A 等
- 多种机房（IDC1-1、数据中心2-1、高机办公楼、土方办公、泵送办公楼、科技园、移动等）

---

## 升级需求与现状 GAP 分析

| 需求项 | 现状 | 差距 |
|--------|------|------|
| 设备字段扩展（到货日期/维保年限/合同号/财务编号等） | 仅有基础字段 | 🔴 缺失 6+ 个关键字段 |
| 在保状态自动计算 | 有维保起止时间但无自动判定 | 🟡 需新增计算逻辑 |
| 设备状态层级化（入库/出库及子状态） | 扁平 status 字符串 | 🔴 需重构为状态机 |
| 入库管理（回收/新购/存放位置/保管员） | 无 | 🔴 全新功能 |
| 出库-报废（备注） | 无 | 🔴 全新功能 |
| 出库-外发（地址/保管人/申请人/项目/业务/部门） | 无 | 🔴 全新功能 |
| 出库-上架（机房/机柜/U位/申请人/项目/业务地址/VIP/管理地址/OS/保管员/备注） | 部分字段已有 | 🟡 需扩展 + 流程化 |
| 出库-下架（自动回收） | 无 | 🔴 全新功能 |
| 审批流程（申请→审批→执行） | 无 | 🔴 全新模块 |
| 机房自定义（定义列/行/机柜高度） | 无，机房仅作为设备字段 | 🔴 全新模块 |
| 2D 机房布局展示图 | 无 | 🔴 全新功能 |
| 机柜设备同步展示 | 有 by-location 查询但无可视化 | 🟡 需前端展示 |
| 入库/出库 Excel 导入导出 | 有基础设备导入导出 | 🟡 需适配新状态流程 |
| 保管员默认值 + 管理员统一修改 | 无 | 🔴 全新功能 |
| 巡检模块 | ✅ 已有 | ✅ 保留，与新功能共存 |

---

## Phase 0: 准备工作与基础设施

### 0.1 数据库迁移策略
- [ ] 设计数据库迁移方案，确保现有数据不丢失
- [ ] 编写迁移脚本：将现有 `Device.Status`（扁平字符串）映射到新的状态体系
- [ ] 备份现有 `dc_manager.db`，建立回滚机制
- [ ] 在 `database/db.go` 中增加版本化迁移支持（而非仅依赖 AutoMigrate）

### 0.2 项目结构调整
- [ ] 后端新增目录结构：
  ```
  backend/
  ├── handlers/
  │   ├── device.go          ← 重构
  │   ├── device_workflow.go  ← 新增：入库/出库/上架/下架操作
  │   ├── approval.go         ← 新增：审批流程
  │   ├── datacenter.go       ← 新增：机房管理
  │   └── ...（保留现有）
  ├── models/
  │   ├── device.go           ← 重构
  │   ├── device_operation.go ← 新增：操作记录
  │   ├── approval.go         ← 新增：审批模型
  │   ├── datacenter.go       ← 新增：机房/机柜模型
  │   └── ...（保留现有）
  └── services/               ← 新增：业务逻辑层
      ├── warranty.go          ← 在保状态计算
      └── workflow.go          ← 状态流转逻辑
  ```
- [ ] 前端新增页面规划：
  ```
  frontend/src/pages/
  ├── DeviceWorkflow.tsx       ← 新增：设备入库/出库操作页
  ├── Approvals.tsx            ← 新增：审批管理页
  ├── DatacenterManage.tsx     ← 新增：机房定义管理页
  ├── DatacenterLayout.tsx     ← 新增：2D机房布局展示页
  └── ...（保留现有）
  ```

---

## Phase 1: 设备模型重构（后端核心）

### 1.1 Device 模型字段扩展
> 文件: `backend/models/device.go`

- [ ] 新增以下字段到 `Device` struct：

| 新字段 | 类型 | 说明 | 对应需求 |
|--------|------|------|----------|
| `Vendor` | string | 厂商 | 需求 1 |
| `ArrivalDate` | *time.Time | 到货日期 | 需求 5 |
| `WarrantyYears` | int | 原厂维保年限 | 需求 7 |
| `ContractNo` | string | 合同号 | 需求 9 |
| `FinanceNo` | string | 财务编号 | 需求 11 |
| `DeviceStatus` | string | 设备主状态：`in_stock` / `out_stock` | 需求 12 |
| `SubStatus` | string | 子状态：`recycled` / `new_purchase` / `scrapped` / `dispatched` / `racked` / `unracked` | 需求 12.x |
| `StorageLocation` | string | 存放位置（入库/下架时） | 需求 12.1.3, 12.2.4.1 |
| `Custodian` | string | 保管员 | 需求 12.1.4 |
| `ScrapRemark` | string | 报废备注 | 需求 12.2.1.1 |
| `DispatchAddress` | string | 外发地址 | 需求 12.2.2.1 |
| `DispatchCustodian` | string | 外发保管人 | 需求 12.2.2.2 |
| `Applicant` | string | 申请人（外发/上架） | 需求 12.2.2.3, 12.2.3.5 |
| `ProjectName` | string | 项目名称 | 需求 12.2.2.4 |
| `BusinessUnit` | string | 所属业务 | 需求 12.2.2.5 |
| `Department` | string | 所属部门 | 需求 12.2.2.6 |
| `UCount` | int | 占用几U（上架时） | 需求 12.2.3.4 |
| `BusinessAddress` | string | 业务地址（上架） | 需求 12.2.3.9 |
| `VipAddress` | string | VIP地址（上架，可选） | 需求 12.2.3.10 |

- [ ] 保留并复用现有字段映射：
  - `Brand` → 对应需求中的「厂商」（或与新 `Vendor` 区分，厂商=总厂商，品牌=品牌线）
  - `Datacenter` → 上架机房（需求 12.2.3.1）
  - `Cabinet` → 上架机柜（需求 12.2.3.2）
  - `StartU` / `UPosition` → 起始U位（需求 12.2.3.3）
  - `MgmtIP` → 管理地址（需求 12.2.3.11）
  - `OS` → 操作系统版本（需求 12.2.3.12）
  - `IPAddress` → 业务地址（需求 12.2.3.9）或保持独立
  - `Owner` → 责任人/保管员

- [ ] **在保状态计算逻辑**（需求 8）：不存库，API 返回时动态计算
  ```go
  // services/warranty.go
  // 在保判定: 起始维保日期 + 原厂维保年限 > 当前日期 → 在保，否则脱保
  func CalcWarrantyStatus(warrantyStart *time.Time, warrantyYears int) string
  ```
  - 在 `GetDevices` / `GetDevice` 响应中附加 `warranty_status` 字段（`in_warranty` / `out_of_warranty`）

### 1.2 设备状态机设计
> 新增文件: `backend/services/workflow.go`

- [ ] 定义合法状态流转规则：
  ```
  [新建] → 入库(新购)
  [新建] → 入库(回收)
  入库(新购/回收) → 出库(上架)
  入库(新购/回收) → 出库(外发)
  入库(新购/回收) → 出库(报废)
  出库(上架) → 出库(下架) → 自动变为 入库(回收)
  ```
- [ ] 状态流转时自动处理：
  - 上架 → 写入机房/机柜/U位等上架字段
  - 下架 → 清空机房/机柜/U位字段，自动设为"入库-回收"，填入存放位置
  - 报废 → 记录报废备注，设备标记为不可再上架
- [ ] 每次状态变更生成操作记录（见 1.3）

### 1.3 设备操作记录模型
> 新增文件: `backend/models/device_operation.go`

- [ ] 新增 `DeviceOperation` 模型：
  ```go
  type DeviceOperation struct {
      ID            uint       `json:"id" gorm:"primaryKey"`
      DeviceID      uint       `json:"device_id" gorm:"index"`
      OperationType string     `json:"operation_type"`  // in_stock_new, in_stock_recycle, rack, unrack, dispatch, scrap
      FromStatus    string     `json:"from_status"`
      ToStatus      string     `json:"to_status"`
      OperatorID    uint       `json:"operator_id"`     // 执行人
      ApprovalID    *uint      `json:"approval_id"`     // 关联审批单
      Details       string     `json:"details"`         // JSON: 操作详细信息
      CreatedAt     time.Time  `json:"created_at"`
  }
  ```
- [ ] 操作记录 API：`GET /api/devices/:id/operations`（查看设备变更历史）

### 1.4 保管员默认值机制
> 需求 12.2.3.13

- [ ] 新增系统配置表 `SystemConfig`：
  ```go
  type SystemConfig struct {
      Key   string `json:"key" gorm:"primaryKey"`
      Value string `json:"value"` // JSON
  }
  ```
- [ ] 配置项 `default_custodians`：JSON 数组 `["某某1", "某某2"]`
- [ ] 上架操作时若未指定保管员，自动填充默认保管员
- [ ] 管理员 API 修改默认保管员：`PUT /api/config/default_custodians`
- [ ] 管理员可批量修改已有设备的保管员：`PUT /api/devices/batch-custodian`

---

## Phase 2: 审批流程模块

### 2.1 审批模型
> 新增文件: `backend/models/approval.go`

- [ ] 新增 `Approval` 模型：
  ```go
  type Approval struct {
      ID             uint       `json:"id" gorm:"primaryKey"`
      ApprovalNo     string     `json:"approval_no" gorm:"uniqueIndex"` // 审批单号，自动生成
      DeviceID       uint       `json:"device_id" gorm:"index"`
      OperationType  string     `json:"operation_type"`   // rack, unrack, dispatch, scrap, in_stock
      RequestData    string     `json:"request_data"`     // JSON: 申请详细数据
      ApplicantID    uint       `json:"applicant_id"`     // 申请人 (User ID)
      ApplicantName  string     `json:"applicant_name"`
      ApproverID     *uint      `json:"approver_id"`      // 审批人
      ApproverName   string     `json:"approver_name"`
      Status         string     `json:"status"`           // pending, approved, rejected, executed, cancelled
      ApproveRemark  string     `json:"approve_remark"`   // 审批意见
      ApprovedAt     *time.Time `json:"approved_at"`
      ExecutedAt     *time.Time `json:"executed_at"`
      CreatedAt      time.Time  `json:"created_at"`
      UpdatedAt      time.Time  `json:"updated_at"`
  }
  ```

### 2.2 审批 API
> 新增文件: `backend/handlers/approval.go`

- [ ] **提交审批**: `POST /api/approvals`
  - 申请人填写操作类型 + 操作数据（如上架信息、外发信息等）
  - 自动生成审批单号（如 `APR-20260412-001`）
  - 创建后状态为 `pending`
- [ ] **审批列表**: `GET /api/approvals`（支持按状态、操作类型、申请人筛选+分页）
- [ ] **审批详情**: `GET /api/approvals/:id`
- [ ] **审批操作**: `PUT /api/approvals/:id/approve`（通过）/ `PUT /api/approvals/:id/reject`（驳回）
  - 审批通过后状态变为 `approved`
- [ ] **执行操作**: `PUT /api/approvals/:id/execute`
  - 仅 `approved` 状态可执行
  - 执行时调用 Phase 1 的状态流转逻辑，实际修改设备状态和字段
  - 执行后状态变为 `executed`
- [ ] **取消审批**: `PUT /api/approvals/:id/cancel`（申请人可取消 `pending` 状态的审批）

### 2.3 权限扩展
> 修改文件: `backend/models/role.go`

- [ ] 新增权限码：
  ```go
  PermApprovalSubmit  = "approval:submit"   // 提交审批申请
  PermApprovalApprove = "approval:approve"  // 审批（通过/驳回）
  PermApprovalExecute = "approval:execute"  // 执行已批准操作
  PermApprovalView    = "approval:view"     // 查看审批记录
  PermDatacenterManage = "datacenter:manage" // 机房管理
  PermDatacenterView   = "datacenter:view"   // 查看机房布局
  PermConfigManage     = "config:manage"     // 系统配置
  ```
- [ ] 更新 `AllPermissions`、`PermissionGroups`、`PermissionLabels`
- [ ] 更新默认角色权限：admin 拥有全部，新增"操作员"角色模板

### 2.4 审批前端页面
> 新增文件: `frontend/src/pages/Approvals.tsx`

- [ ] 审批列表页：Tab 切换（待我审批 / 我的申请 / 全部）
- [ ] 审批详情抽屉/Modal：显示申请内容、设备信息、审批意见输入
- [ ] 审批操作按钮：通过 / 驳回 / 执行 / 取消
- [ ] 在侧边栏新增"审批管理"菜单项（带待审批数量 Badge）

---

## Phase 3: 机房管理模块

### 3.1 机房数据模型
> 新增文件: `backend/models/datacenter.go`

- [ ] `Datacenter` 模型（机房定义）：
  ```go
  type Datacenter struct {
      ID        uint   `json:"id" gorm:"primaryKey"`
      Name      string `json:"name" gorm:"uniqueIndex;not null"` // 如"数据中心1-1"、"土方办公楼机房"
      Remark    string `json:"remark"`
      CreatedAt time.Time `json:"created_at"`
      UpdatedAt time.Time `json:"updated_at"`
  }
  ```

- [ ] `CabinetColumn` 模型（列定义）：
  ```go
  type CabinetColumn struct {
      ID           uint   `json:"id" gorm:"primaryKey"`
      DatacenterID uint   `json:"datacenter_id" gorm:"index;not null"`
      Name         string `json:"name" gorm:"not null"`   // 如 A、B、C、HDA、PDU、空调
      SortOrder    int    `json:"sort_order"`              // 显示排序
      ColumnType   string `json:"column_type"`             // cabinet / hda / pdu / aircon / other
  }
  ```

- [ ] `CabinetRow` 模型（行定义）：
  ```go
  type CabinetRow struct {
      ID           uint   `json:"id" gorm:"primaryKey"`
      DatacenterID uint   `json:"datacenter_id" gorm:"index;not null"`
      Name         string `json:"name"`          // 如 1、2、3、"C-A"、空白
      SortOrder    int    `json:"sort_order"`
  }
  ```

- [ ] `Cabinet` 模型（机柜）：
  ```go
  type Cabinet struct {
      ID           uint   `json:"id" gorm:"primaryKey"`
      DatacenterID uint   `json:"datacenter_id" gorm:"index;not null"`
      ColumnID     uint   `json:"column_id"`
      RowID        uint   `json:"row_id"`
      Name         string `json:"name" gorm:"not null"`   // 如 "A-01"，自动或手动生成
      Height       int    `json:"height" gorm:"default:42"` // 机柜高度 U 数
      Width        int    `json:"width" gorm:"default:60"`  // 宽 cm
      Depth        int    `json:"depth" gorm:"default:120"` // 深 cm
      CabinetType  string `json:"cabinet_type"`             // standard / hda / pdu / aircon
      Remark       string `json:"remark"`
  }
  ```

### 3.2 机房管理 API
> 新增文件: `backend/handlers/datacenter.go`

- [ ] **机房 CRUD**:
  - `GET /api/datacenters` — 列表
  - `POST /api/datacenters` — 创建机房
  - `PUT /api/datacenters/:id` — 编辑
  - `DELETE /api/datacenters/:id` — 删除（需检查是否有关联设备）

- [ ] **列/行定义**:
  - `GET /api/datacenters/:id/columns` — 获取列定义
  - `POST /api/datacenters/:id/columns` — 批量设置列（传入数组，一次覆盖）
  - `GET /api/datacenters/:id/rows` — 获取行定义
  - `POST /api/datacenters/:id/rows` — 批量设置行

- [ ] **机柜管理**:
  - `GET /api/datacenters/:id/cabinets` — 获取机房下所有机柜
  - `POST /api/datacenters/:id/cabinets/generate` — 根据列×行自动生成机柜
  - `PUT /api/cabinets/:id` — 编辑单个机柜（高度、类型等）
  - `GET /api/cabinets/:id/devices` — 获取机柜内设备（含 U 位占用）

- [ ] **机房布局数据（供前端渲染）**:
  - `GET /api/datacenters/:id/layout` — 返回完整布局数据：
    ```json
    {
      "datacenter": {...},
      "columns": [...],
      "rows": [...],
      "cabinets": [
        {
          "id": 1, "name": "A-01", "column": "A", "row": "1",
          "height": 47, "devices": [
            {"start_u": 3, "end_u": 4, "device_id": 123, "brand": "华为", "model": "RH2288V2", ...}
          ]
        }
      ]
    }
    ```

### 3.3 机房数据关联
- [ ] 重构 Device 的 `Datacenter` / `Cabinet` 字段：
  - 新增 `DatacenterID` (uint, FK) 和 `CabinetID` (uint, FK) 关联到机房管理模块
  - 保留原 `Datacenter` / `Cabinet` 字符串字段作为兼容（渐进迁移）
  - 上架操作时优先使用 ID 关联，同时回写字符串字段
- [ ] 编写数据迁移脚本：根据现有设备的 `datacenter` + `cabinet` 字符串，匹配并关联到新建的机房/机柜记录

### 3.4 机房管理前端
> 新增文件: `frontend/src/pages/DatacenterManage.tsx`

- [ ] 机房列表：左侧机房树 + 右侧配置区
- [ ] 机房配置表单：
  - 基本信息（名称、备注）
  - 列定义：可拖拽排序的列表，支持添加/删除/重命名列，可选列类型（普通机柜/HDA/PDU/空调）
  - 行定义：同上，支持空白行
  - 机柜高度：统一设置或逐柜设置（默认 42U）
- [ ] 机柜生成：配置完列和行后一键生成机柜网格
- [ ] 机柜编辑：点击单个机柜可修改名称、高度、备注

### 3.5 2D 机房布局展示页
> 新增文件: `frontend/src/pages/DatacenterLayout.tsx`

- [ ] 顶部机房选择器（下拉切换不同机房）
- [ ] 2D 网格布局渲染（基于 Canvas 或 SVG）：
  - 按列×行排列机柜方块
  - 机柜尺寸按比例：宽 60cm / 深 120cm
  - 行与行之间间隔 120cm（需求 17.2）
  - 同列机柜相邻（需求 17.3）
  - 非机柜列（HDA/PDU/空调）用不同颜色/图标表示
- [ ] 机柜交互：
  - 悬浮显示机柜名称 + 使用率（已用 U 数 / 总 U 数）
  - 点击机柜展开侧边抽屉，显示该机柜内设备的 U 位分布图（纵向柱状）
  - U 位分布图中设备块颜色按品牌或状态区分
  - 点击设备块可查看设备详情
- [ ] 颜色图例：空闲(绿)/已用(蓝)/告警(红)/非机柜(灰)
- [ ] 布局统计卡片：总机柜数、已用机柜数、总 U 位数、已用 U 位数、使用率

---

## Phase 4: 设备入库/出库操作流程（前端）

### 4.1 设备操作入口重构
> 修改文件: `frontend/src/pages/Devices.tsx`

- [ ] 设备列表新增列：
  - "在保状态" 列（在保=绿色标签 / 脱保=红色标签），根据 API 返回的 `warranty_status` 显示
  - "设备状态" 列改为显示 `主状态-子状态`（如"入库-新购"、"出库-上架"）
  - "合同号"、"财务编号" 列
- [ ] 设备编辑 Modal 增加新字段
- [ ] 新增操作按钮组（根据当前设备状态动态显示可用操作）：
  - 入库状态 → 可操作：上架 / 外发 / 报废
  - 出库-上架状态 → 可操作：下架
  - 每个操作点击后弹出对应表单 Modal

### 4.2 入库操作表单
- [ ] 入库类型选择：新购 / 回收
- [ ] 新购表单：存放位置、保管员（默认值自动填充）
- [ ] 回收表单：存放位置、保管员
- [ ] 提交后创建审批申请（调用 `POST /api/approvals`）

### 4.3 出库-上架操作表单
- [ ] 表单字段（需求 12.2.3）：
  - 机房（下拉选择，数据来自机房管理模块）
  - 机柜（联动下拉，根据选择的机房筛选）
  - 起始U位（数字输入）
  - 占用几U（数字输入）
  - 申请人
  - 项目名称
  - 所属业务
  - 所属部门
  - 业务地址
  - VIP地址（可选）
  - 管理地址
  - 操作系统版本
  - 保管员（默认值自动填充，可修改）
  - 备注
- [ ] U 位冲突检测：提交前 API 校验目标 U 位是否已被占用
- [ ] 提交后创建审批申请

### 4.4 出库-外发操作表单
- [ ] 表单字段（需求 12.2.2）：地址、保管人、申请人、项目名称、所属业务、所属部门
- [ ] 提交后创建审批申请

### 4.5 出库-报废操作表单
- [ ] 表单字段：报废备注
- [ ] 提交后创建审批申请

### 4.6 出库-下架操作表单
- [ ] 表单字段：存放位置
- [ ] 提示："确认下架后设备将自动变为入库-回收状态"
- [ ] 提交后创建审批申请
- [ ] 审批执行后自动：清空上架字段 → 设为入库-回收 → 填入存放位置

### 4.7 设备操作历史
- [ ] 设备详情页/抽屉新增"操作记录" Tab
- [ ] 时间线展示所有状态变更记录（含操作人、时间、审批单号、详情）

---

## Phase 5: 导入导出功能增强

### 5.1 入库导入导出（需求 12.1.3）
- [ ] 入库导入模板：包含设备基础字段 + 入库类型 + 存放位置 + 保管员
- [ ] 导入时自动设置设备状态为"入库-新购"或"入库-回收"
- [ ] 导入导出按钮放置在入库管理视图中

### 5.2 出库导入导出（需求 12.2.5）
- [ ] 出库导出：导出当前已出库设备列表（含上架/外发/报废信息）
- [ ] 上架批量导入：Excel 包含机房、机柜、U位等上架信息，批量操作
- [ ] 导入时执行 U 位冲突批量校验

### 5.3 现有导入逻辑适配
> 修改文件: `backend/handlers/device.go` - `ImportDevices()`

- [ ] 导入 Excel 列映射增加新字段（到货日期、合同号、财务编号、维保年限等）
- [ ] 导入时根据现有台账数据文件的 Sheet 结构智能识别（兼容现有 "数据中心"、"存储" 等 Sheet 的不同列结构）

---

## Phase 6: Dashboard 升级

### 6.1 大屏数据扩展
> 修改文件: `backend/handlers/inspection.go`（Dashboard handler 在此文件）

- [ ] 新增统计接口或扩展 `GET /api/dashboard`：
  - 设备总数 / 在保设备数 / 脱保设备数
  - 各状态设备数量（入库-新购、入库-回收、出库-上架、出库-外发、出库-报废）
  - 各机房设备数量 + U 位使用率
  - 各厂商/设备类型设备数量
  - 维保即将到期设备列表（30天/60天/90天内到期）
  - 待审批数量
  - 保留现有巡检统计数据

### 6.2 Dashboard 前端升级
> 修改文件: `frontend/src/pages/Dashboard.tsx`

- [ ] 新增 Tab 切换：设备总览 / 巡检大屏
- [ ] 设备总览 Tab：
  - 统计卡片：设备总数、在保数、脱保数、待审批数
  - 设备状态分布饼图
  - 各机房设备数量柱状图（复用现有，改用新机房管理数据）
  - 维保到期预警列表
  - 设备类型/厂商分布图
  - 机房 U 位使用率横向柱状图
- [ ] 巡检大屏 Tab：保持现有巡检统计内容

---

## Phase 7: 前端导航与菜单调整

### 7.1 侧边栏菜单重构
> 修改文件: `frontend/src/App.tsx`

- [ ] 新菜单结构：
  ```
  📊 总览大屏（Dashboard）
  📦 设备管理
     └ 设备台账（现有，增强）
     └ 入库管理（新增筛选视图）
     └ 出库管理（新增筛选视图）
  ✅ 审批管理（新增）
  🏢 机房管理
     └ 机房配置（新增）
     └ 机房布局（新增，2D展示）
  🔍 巡检记录（保留现有）
  👥 用户管理（保留现有）
  🔐 角色管理（保留现有）
  ⚙️ 系统配置（新增：默认保管员等）
  ```
- [ ] 使用 Ant Design Menu 的 SubMenu 实现二级菜单
- [ ] 移动端底部导航适配：保留核心入口（总览/设备/审批/机房），其余收入更多菜单

### 7.2 系统配置页
> 新增文件: `frontend/src/pages/SystemConfig.tsx`

- [ ] 默认保管员设置（可添加/删除，支持排序）
- [ ] 其他全局配置预留

---

## Phase 8: 数据迁移与兼容

### 8.1 现有数据迁移脚本
> 新增: `backend/cmd/migrate/main.go`

- [ ] **设备状态迁移**：
  - 现有 `Status` 字段值（如 Online/Offline 等）→ 映射到新的 DeviceStatus + SubStatus
  - 已有 datacenter + cabinet + start_u 的设备 → 设为"出库-上架"
  - 无位置信息的设备 → 设为"入库-新购"
- [ ] **机房数据初始化**：
  - 从现有设备数据中提取去重的 datacenter 值，自动创建 Datacenter 记录
  - 从 Excel 落位图 Sheet（1-1机房/1-2机房/2-1机房）解析列名和机柜编号，自动创建列/行/机柜记录
- [ ] **字段迁移**：
  - `Brand` → 同时写入 `Vendor`（首次迁移）
  - 计算 `WarrantyYears`：从现有 warranty_start 和 warranty_end 推算

### 8.2 兼容性保障
- [ ] 现有 API 接口保持向后兼容，新字段 optional
- [ ] MCP server（`backend/mcp/server.go`）同步更新：新增工具方法适配新模型
- [ ] 导入功能兼容现有 Excel 格式（无新字段时使用默认值）

---

## Phase 9: 测试与部署

### 9.1 后端测试
- [ ] 状态流转单元测试（合法/非法流转）
- [ ] 审批流程集成测试（提交→审批→执行→设备状态变更）
- [ ] U 位冲突检测测试
- [ ] 在保状态计算测试（边界日期）
- [ ] 数据迁移脚本在测试库上验证

### 9.2 前端测试
- [ ] 设备操作表单验证
- [ ] 审批流程页面交互
- [ ] 2D 机房布局渲染（多种机房配置）
- [ ] 移动端适配测试

### 9.3 部署
- [ ] 更新 Docker Compose 配置（如有新服务）
- [ ] 更新 Dockerfile 依赖
- [ ] 编写升级部署文档（含数据迁移步骤）
- [ ] 生产环境备份 + 迁移 + 验证流程

---

## 实施优先级建议

| 优先级 | Phase | 说明 | 预估复杂度 |
|--------|-------|------|-----------|
| P0 | Phase 0 | 准备工作 | ⭐⭐ |
| P0 | Phase 1 | 设备模型重构（后续一切基础） | ⭐⭐⭐⭐ |
| P1 | Phase 2 | 审批流程（用户明确要求） | ⭐⭐⭐⭐ |
| P1 | Phase 4 | 入库/出库操作前端（核心交互） | ⭐⭐⭐⭐ |
| P2 | Phase 3 | 机房管理模块 | ⭐⭐⭐⭐⭐ |
| P2 | Phase 5 | 导入导出增强 | ⭐⭐⭐ |
| P3 | Phase 6 | Dashboard 升级 | ⭐⭐⭐ |
| P3 | Phase 7 | 菜单与配置页 | ⭐⭐ |
| P4 | Phase 8 | 数据迁移 | ⭐⭐⭐ |
| P4 | Phase 9 | 测试与部署 | ⭐⭐⭐ |

---

## 关键技术决策备忘

1. **数据库**: 继续使用 SQLite + GORM AutoMigrate，适合当前规模。若未来设备量超万级考虑 PostgreSQL。
2. **2D 布局**: 推荐使用 React + SVG（或 Canvas）自行绘制，比引入重型库更可控。ECharts 不适合此场景。
3. **审批流程**: 当前先做单级审批（一个审批人），不做多级会签。预留 `ApproverID` 可扩展。
4. **状态机**: 在 Go 层实现硬编码规则，不引入额外状态机库。状态少（<10种），不需要工作流引擎。
5. **巡检模块**: 保留现有全部代码和 API，仅在 Dashboard 做 Tab 分离。
6. **MCP**: 升级完成后同步更新 MCP tool 定义，暴露新的设备操作和机房查询能力。
7. **3D 展示**: 本期不实施。Phase 3.5 的 2D 布局完成后，后续可用 Three.js 升级（在 2D 数据模型基础上添加渲染层即可）。
