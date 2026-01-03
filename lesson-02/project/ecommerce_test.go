package project

import (
	"context"
	"errors"
	"fmt"
	"gohomeworklesson02/testutil"
	"math/rand"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// 业务错误定义
// 使用自定义错误类型便于在业务层进行错误判断和处理
// 使用 errors.Is 可以判断是否为特定业务错误
var (
	errNoItems          = errors.New("order must contain at least one item")
	errOutOfStock       = errors.New("product stock is insufficient")
	errOrderAlreadyPaid = errors.New("order already paid")
)

// User 用户模型
// 电商系统的用户实体，包含用户基本信息和时间戳
// 索引设计：
//   - Email 使用 uniqueIndex 确保邮箱唯一性，同时提供快速查询
//   - 如果后续需要按名称搜索，可以添加 Name 字段的普通索引
type User struct {
	ID        uint      `gorm:"primaryKey"`                    // 主键，自增
	Email     string    `gorm:"size:128;uniqueIndex;not null"` // 邮箱，唯一索引，非空
	Name      string    `gorm:"size:64;not null"`              // 用户名，非空
	CreatedAt time.Time // 创建时间，GORM 自动管理
	UpdatedAt time.Time // 更新时间，GORM 自动管理
}

// Product 商品模型
// 电商系统的商品实体，包含商品信息和库存
// 索引设计：
//   - SKU 使用 uniqueIndex 确保商品编码唯一性
//   - Name 可以添加普通索引用于商品名称搜索（示例中未添加，实际项目中建议添加）
//
// 注意：
//   - Price 使用 int64 存储，单位为分（避免浮点数精度问题）
//   - Stock 需要在事务中锁定更新，防止并发扣减导致超卖
type Product struct {
	ID        uint      `gorm:"primaryKey"`                   // 主键，自增
	Name      string    `gorm:"size:128;not null"`            // 商品名称，非空
	SKU       string    `gorm:"size:32;uniqueIndex;not null"` // 商品编码（Stock Keeping Unit），唯一索引，非空
	Price     int64     `gorm:"not null"`                     // 价格（单位：分），非空
	Stock     int       `gorm:"not null"`                     // 库存数量，非空，需要在事务中锁定更新
	CreatedAt time.Time // 创建时间，GORM 自动管理
	UpdatedAt time.Time // 更新时间，GORM 自动管理
}

// Order 订单模型
// 电商系统的订单实体，包含订单信息和关联的订单项
// 索引设计：
//   - OrderNo 使用 uniqueIndex 确保订单号唯一性，支持幂等设计
//   - UserID 使用普通索引，用于查询用户的订单列表
//   - Status 使用普通索引，用于按状态筛选订单（如查询待支付订单）
//   - PaidAt 使用普通索引，用于查询已支付订单的时间范围
//
// 关联关系：
//   - Items: Has Many OrderItem（一个订单包含多个订单项）
//
// 状态枚举：
//   - PENDING: 待支付
//   - PAID: 已支付
//   - CANCELLED: 已取消
type Order struct {
	ID          uint        `gorm:"primaryKey"`                   // 主键，自增
	OrderNo     string      `gorm:"size:32;uniqueIndex;not null"` // 订单号，唯一索引，非空（用于幂等设计）
	UserID      uint        `gorm:"index;not null"`               // 用户ID，普通索引，非空（外键关联 User）
	TotalAmount int64       `gorm:"not null"`                     // 订单总金额（单位：分），非空
	Status      string      `gorm:"size:16;index;not null"`       // 订单状态，普通索引，非空（PENDING/PAID/CANCELLED）
	PaidAt      *time.Time  `gorm:"index"`                        // 支付时间，普通索引，可为空（指针类型表示可选）
	Items       []OrderItem // Has Many: 订单项关联
	CreatedAt   time.Time   // 创建时间，GORM 自动管理
	UpdatedAt   time.Time   // 更新时间，GORM 自动管理
}

// OrderItem 订单项模型
// 订单中的商品明细，记录每个商品的购买数量和单价
// 索引设计：
//   - OrderID 使用普通索引，用于查询订单的所有订单项
//   - ProductID 使用普通索引，用于统计商品的销售情况
//   - 如果需要按商品和时间范围查询，可以添加复合索引 (product_id, created_at)
//
// 关联关系：
//   - Belongs To Order: 订单项属于一个订单
//   - Belongs To Product: 订单项关联一个商品（用于预加载商品信息）
//
// 注意：
//   - UnitPrice 存储下单时的商品单价（快照），避免商品价格变更影响历史订单
//   - Quantity 和 UnitPrice 的乘积应该等于订单总金额的一部分
type OrderItem struct {
	ID        uint      `gorm:"primaryKey"`     // 主键，自增
	OrderID   uint      `gorm:"index;not null"` // 订单ID，普通索引，非空（外键关联 Order）
	ProductID uint      `gorm:"index;not null"` // 商品ID，普通索引，非空（外键关联 Product）
	Product   Product   // Belongs To: 商品关联（用于预加载商品信息）
	Quantity  int       `gorm:"not null"` // 购买数量，非空
	UnitPrice int64     `gorm:"not null"` // 单价（单位：分），非空（下单时的价格快照）
	CreatedAt time.Time // 创建时间，GORM 自动管理
	UpdatedAt time.Time // 更新时间，GORM 自动管理
}

// OrderItemInput 订单项输入结构
// 用于创建订单时的输入参数，不直接映射到数据库表
type OrderItemInput struct {
	ProductID uint // 商品ID
	Quantity  int  // 购买数量
}

// SalesSummary 销售报表汇总结构
// 用于存储按日期聚合的销售数据，不直接映射到数据库表
// 通过聚合查询（GROUP BY）和 Scan 方法填充数据
type SalesSummary struct {
	Day         string // 日期（格式：YYYY-MM-DD）
	OrderCount  int64  // 订单数量
	ItemCount   int64  // 商品数量（所有订单项的数量总和）
	TotalAmount int64  // 销售总额（单位：分）
}

// TestEcommerceFlow 电商系统完整流程测试
// 本测试演示了电商系统的核心业务流程：
// 1. 数据模型初始化（AutoMigrate）
// 2. 初始化数据（用户、商品）
// 3. 下单流程（库存校验、扣减、生成订单）
// 4. 库存不足场景处理
// 5. 订单支付流程
// 6. 订单查询与预加载
// 7. 销售报表聚合查询
func TestEcommerceFlow(t *testing.T) {
	ctx := context.Background()
	// 使用 testutil.NewTestDB 创建测试数据库
	// 支持 SQLite、MySQL、PostgreSQL，可通过环境变量配置
	db := testutil.NewTestDB(t, "ecommerce.db")

	// 数据模型初始化：AutoMigrate 自动创建表结构
	// 如果表已存在，会根据模型定义更新表结构（添加新字段、索引等）
	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// 初始化测试数据：如果表为空，则插入初始数据
	// 使用 Count 检查避免重复插入
	if err := seedData(db); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	// 展示初始库存状态
	t.Log("== 初始库存 ==")
	products := fetchProducts(t, db)
	for _, p := range products {
		t.Logf(" - %s 库存:%d 价格:%.2f", p.Name, p.Stock, float64(p.Price)/100)
	}

	// 下单流程：创建包含多个商品的订单
	// CreateOrder 内部使用事务保证数据一致性
	t.Log("== 下单流程 ==")
	order, err := CreateOrder(ctx, db, 1, []OrderItemInput{
		{ProductID: products[0].ID, Quantity: 1},
		{ProductID: products[1].ID, Quantity: 2},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	// 打印订单详情（包含预加载的商品信息）
	logOrder(t, db, order.OrderNo)

	// 验证库存扣减：查询更新后的库存
	t.Log("== 库存回查 ==")
	updated := fetchProducts(t, db)
	for _, p := range updated {
		t.Logf(" - %s 库存:%d", p.Name, p.Stock)
	}

	// 库存不足场景：尝试购买超过库存数量的商品
	// 应该返回 errOutOfStock 错误
	t.Log("== 库存不足场景 ==")
	_, err = CreateOrder(ctx, db, 1, []OrderItemInput{
		{ProductID: products[1].ID, Quantity: 100},
	})
	// 使用 errors.Is 判断是否为特定业务错误
	if !errors.Is(err, errOutOfStock) {
		t.Fatalf("expected out of stock, got %v", err)
	}

	// 订单支付流程：标记订单为已支付
	t.Log("== 标记订单支付 ==")
	if err := MarkOrderPaid(ctx, db, order.OrderNo); err != nil {
		t.Fatalf("mark paid: %v", err)
	}
	// 再次打印订单详情，验证状态和支付时间已更新
	logOrder(t, db, order.OrderNo)

	// 订单总览：查询所有订单（包含预加载的订单项和商品信息）
	t.Log("== 订单总览 ==")
	orders := fetchOrders(t, db)
	for _, o := range orders {
		t.Logf("- %s [%s] items=%d", o.OrderNo, o.Status, len(o.Items))
	}

	// 销售报表：按日期聚合已支付订单的销售数据
	t.Log("== 销售报表 ==")
	report, err := SalesReport(db)
	if err != nil {
		t.Fatalf("sales report: %v", err)
	}
	for _, row := range report {
		t.Logf("%s -> 订单:%d 商品:%d 销售额:%.2f",
			row.Day, row.OrderCount, row.ItemCount, float64(row.TotalAmount)/100)
	}
}

// CreateOrder 创建订单
// 这是电商系统的核心业务函数，实现了完整的下单流程
// 流程步骤：
//  1. 校验订单项不为空
//  2. 在事务中执行以下操作（保证原子性）：
//     a. 加载用户信息
//     b. 锁定并加载商品信息（使用 FOR UPDATE 防止并发问题）
//     c. 校验库存是否充足
//     d. 扣减库存（使用 UpdateColumn 直接更新，避免零值问题）
//     e. 计算订单总金额
//     f. 生成订单号（幂等设计）
//     g. 创建订单和订单项
//  3. 如果任何步骤失败，事务自动回滚
//
// 关键设计点：
// - 使用事务保证数据一致性（库存扣减和订单创建要么全部成功，要么全部失败）
// - 使用 FOR UPDATE 锁定商品记录，防止并发下单导致超卖
// - 订单号唯一索引确保幂等性（重复下单会失败）
// - 使用自定义错误类型便于业务层判断错误类型
func CreateOrder(ctx context.Context, db *gorm.DB, userID uint, items []OrderItemInput) (*Order, error) {
	// 校验：订单必须包含至少一个商品
	if len(items) == 0 {
		return nil, errNoItems
	}

	var order Order
	// 使用事务包装整个下单流程
	// Transaction 方法会在函数返回 error 时自动回滚，返回 nil 时自动提交
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 步骤1: 加载用户信息
		var user User
		if err := tx.First(&user, userID).Error; err != nil {
			return fmt.Errorf("load user: %w", err)
		}

		// 步骤2: 收集需要查询的商品ID
		productIDs := make([]uint, 0, len(items))
		for _, item := range items {
			productIDs = append(productIDs, item.ProductID)
		}

		// 步骤3: 锁定并加载商品信息
		// clause.Locking{Strength: "UPDATE"} 相当于 SQL 的 SELECT ... FOR UPDATE
		// 这会锁定查询到的商品记录，防止其他事务同时修改库存
		// 锁定会持续到事务结束（提交或回滚）
		var products []Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ?", productIDs).
			Find(&products).Error; err != nil {
			return fmt.Errorf("load products: %w", err)
		}

		// 步骤4: 构建商品ID到商品对象的映射，便于快速查找
		productMap := make(map[uint]Product, len(products))
		for _, p := range products {
			productMap[p.ID] = p
		}

		// 步骤5: 校验库存并扣减，同时计算订单总金额
		var total int64
		orderItems := make([]OrderItem, 0, len(items))

		for _, item := range items {
			// 校验商品是否存在
			p, ok := productMap[item.ProductID]
			if !ok {
				return fmt.Errorf("product %d not found", item.ProductID)
			}
			// 校验购买数量是否有效
			if item.Quantity <= 0 {
				return fmt.Errorf("invalid quantity for product %d", item.ProductID)
			}
			// 校验库存是否充足
			if p.Stock < item.Quantity {
				// 使用 %w 包装错误，保留错误链，便于使用 errors.Is 判断
				return fmt.Errorf("%w: %s (需要 %d, 当前 %d)", errOutOfStock, p.Name, item.Quantity, p.Stock)
			}

			// 扣减库存：使用 UpdateColumn 直接更新，避免零值问题
			// gorm.Expr 允许使用 SQL 表达式，这里使用 stock - ? 原子性扣减
			// 注意：由于已经使用 FOR UPDATE 锁定，这里不会出现并发问题
			if err := tx.Model(&Product{}).
				Where("id = ?", p.ID).
				UpdateColumn("stock", gorm.Expr("stock - ?", item.Quantity)).Error; err != nil {
				return fmt.Errorf("update stock: %w", err)
			}

			// 计算订单项金额并累加到总金额
			line := int64(item.Quantity) * p.Price
			total += line
			// 构建订单项（不包含 ID，由 GORM 自动生成）
			orderItems = append(orderItems, OrderItem{
				ProductID: p.ID,
				Quantity:  item.Quantity,
				UnitPrice: p.Price, // 保存下单时的价格快照
			})
		}

		// 步骤6: 生成订单号并创建订单
		// 订单号使用唯一索引，确保幂等性（重复下单会因唯一约束失败）
		order = Order{
			OrderNo:     generateOrderNo(), // 生成唯一订单号
			UserID:      user.ID,
			TotalAmount: total,
			Status:      "PENDING",  // 初始状态为待支付
			Items:       orderItems, // 关联的订单项，GORM 会自动创建
		}

		// 创建订单：GORM 会自动创建关联的订单项（因为 Items 字段已填充）
		if err := tx.Create(&order).Error; err != nil {
			return fmt.Errorf("create order: %w", err)
		}

		// 返回 nil 表示事务成功，GORM 会自动提交
		return nil
	})

	// 如果事务失败，返回错误
	if err != nil {
		return nil, err
	}
	// 事务成功，返回创建的订单
	return &order, nil
}

// MarkOrderPaid 标记订单为已支付
// 支付流程的核心函数，负责更新订单状态和支付时间
// 流程步骤：
// 1. 在事务中锁定并加载订单（使用 FOR UPDATE 防止并发支付）
// 2. 校验订单状态（防止重复支付）
// 3. 更新订单状态为 PAID 并记录支付时间
//
// 关键设计点：
// - 使用事务保证原子性
// - 使用 FOR UPDATE 锁定订单，防止并发支付导致的状态不一致
// - 校验订单状态，实现幂等性（重复支付会返回错误）
// - 可扩展：可以在这里添加扣减用户余额、写入支付日志等操作
func MarkOrderPaid(ctx context.Context, db *gorm.DB, orderNo string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order Order
		// 锁定并加载订单：使用 FOR UPDATE 防止并发支付
		// 如果多个请求同时支付同一订单，只有一个会成功
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_no = ?", orderNo).
			First(&order).Error; err != nil {
			return fmt.Errorf("load order: %w", err)
		}

		// 幂等性检查：如果订单已经支付，返回错误
		// 这防止了重复支付的问题
		if order.Status == "PAID" {
			return errOrderAlreadyPaid
		}

		// 更新订单状态和支付时间
		// 使用 map[string]any 可以更新指定字段，忽略零值
		// paid_at 使用指针类型，可以设置为 nil（取消支付）或具体时间
		now := time.Now()
		return tx.Model(&order).Updates(map[string]any{
			"status":  "PAID",
			"paid_at": &now,
		}).Error
	})
}

// SalesReport 销售报表
// 按日期聚合已支付订单的销售数据
// 返回每天的订单数量、商品数量和销售总额
//
// 查询逻辑：
// 1. 使用 Table 指定主表（orders）
// 2. 使用 Joins 关联订单项表（order_items）
// 3. 使用 Where 过滤已支付订单
// 4. 使用 Select 指定聚合字段：
//   - strftime: SQLite 日期格式化函数（MySQL 使用 DATE，PostgreSQL 使用 TO_CHAR）
//   - COUNT(DISTINCT): 统计不重复的订单数量
//   - SUM: 统计商品数量和销售总额
//
// 5. 使用 Group 按日期分组
// 6. 使用 Order 按日期升序排序
// 7. 使用 Scan 将结果映射到 SalesSummary 结构体
//
// 注意：
// - 本示例使用 SQLite 的 strftime 函数，如果使用 MySQL 或 PostgreSQL，需要调整日期格式化函数
// - 聚合查询必须使用 Scan 而不是 Find（因为结果不直接映射到模型）
func SalesReport(db *gorm.DB) ([]SalesSummary, error) {
	var rows []SalesSummary
	err := db.Table("orders").
		// Select 指定要查询的字段和聚合函数
		// strftime('%Y-%m-%d', ...) 是 SQLite 的日期格式化函数
		// COUNT(DISTINCT ...) 统计不重复的订单数量
		// SUM(...) 统计商品数量和销售总额
		Select(`
			strftime('%Y-%m-%d', orders.created_at) AS day,
			COUNT(DISTINCT orders.id) AS order_count,
			SUM(order_items.quantity) AS item_count,
			SUM(order_items.quantity * order_items.unit_price) AS total_amount`).
		// Joins 关联订单项表，用于统计商品数量和计算销售总额
		Joins("JOIN order_items ON order_items.order_id = orders.id").
		// Where 只统计已支付的订单
		Where("orders.status = ?", "PAID").
		// Group 按日期分组，将同一天的订单聚合在一起
		Group("day").
		// Order 按日期升序排序
		Order("day ASC").
		// Scan 将聚合结果映射到 SalesSummary 结构体
		// 注意：聚合查询必须使用 Scan 而不是 Find
		Scan(&rows).Error
	return rows, err
}

// migrate 数据库迁移
// 使用 AutoMigrate 自动创建或更新表结构
// AutoMigrate 会根据模型定义：
// - 创建不存在的表
// - 添加新的字段和索引
// - 不会删除已存在的字段（安全设计）
// 注意：所有相关的模型必须一起迁移，确保外键关系正确创建
func migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Product{}, &Order{}, &OrderItem{})
}

// seedData 初始化测试数据
// 如果表为空，则插入初始数据（用户和商品）
// 使用 Count 检查避免重复插入，支持多次运行测试
func seedData(db *gorm.DB) error {
	// 初始化用户数据
	var count int64
	if err := db.Model(&User{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		users := []User{
			{Name: "Alice", Email: "alice@example.com"},
			{Name: "Bob", Email: "bob@example.com"},
		}
		if err := db.Create(&users).Error; err != nil {
			return err
		}
	}

	// 初始化商品数据
	if err := db.Model(&Product{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		products := []Product{
			{Name: "Go 语言权威指南", SKU: "BOOK-001", Price: 13800, Stock: 50},
			{Name: "机械键盘", SKU: "GEAR-201", Price: 39900, Stock: 20},
			{Name: "人体工学鼠标", SKU: "GEAR-301", Price: 22900, Stock: 15},
		}
		if err := db.Create(&products).Error; err != nil {
			return err
		}
	}
	return nil
}

// generateOrderNo 生成订单号
// 订单号格式：ORD-YYYYMMDD-XXXX
// - ORD: 订单前缀
// - YYYYMMDD: 日期（8位）
// - XXXX: 随机数（4位，0-9999）
// 注意：订单号使用唯一索引，确保唯一性
// 虽然随机数可能重复，但结合日期后重复概率极低
// 实际项目中建议使用 UUID 或雪花算法生成唯一ID
func generateOrderNo() string {
	return fmt.Sprintf("ORD-%s-%04d", time.Now().Format("20060102"), rand.Intn(10000))
}

// fetchProducts 查询所有商品
// 辅助函数，用于测试中查询商品列表
// 按 ID 升序排序，确保结果顺序一致
func fetchProducts(t *testing.T, db *gorm.DB) []Product {
	t.Helper()
	var products []Product
	if err := db.Order("id").Find(&products).Error; err != nil {
		t.Fatalf("list products: %v", err)
	}
	return products
}

// fetchOrders 查询所有订单（包含预加载的订单项和商品信息）
// 使用 Preload 预加载关联数据，避免 N+1 查询问题
// Preload("Items.Product") 会：
// 1. 先查询所有订单
// 2. 再查询这些订单的所有订单项
// 3. 最后查询这些订单项关联的商品信息
// 这样只需要 3 次 SQL 查询，而不是 N+1 次（N 为订单数量）
func fetchOrders(t *testing.T, db *gorm.DB) []Order {
	t.Helper()
	var orders []Order
	// Preload("Items.Product"): 预加载订单项及其关联的商品
	// 这是嵌套预加载：Order -> OrderItem -> Product
	if err := db.Preload("Items.Product").Order("created_at desc").Find(&orders).Error; err != nil {
		t.Fatalf("list orders: %v", err)
	}
	return orders
}

// logOrder 打印订单详情
// 辅助函数，用于测试中打印订单的详细信息
// 使用 Preload 预加载订单项和商品信息，避免 N+1 查询问题
func logOrder(t *testing.T, db *gorm.DB, orderNo string) {
	t.Helper()
	var order Order
	// Preload("Items.Product"): 预加载订单项及其关联的商品
	// First: 根据订单号查询订单
	if err := db.Preload("Items.Product").
		First(&order, "order_no = ?", orderNo).Error; err != nil {
		t.Fatalf("query order: %v", err)
	}
	// 打印订单基本信息
	t.Logf("订单 %s 状态:%s 总额:%.2f", order.OrderNo, order.Status, float64(order.TotalAmount)/100)
	// 打印订单项详情（商品信息已通过 Preload 加载）
	for _, item := range order.Items {
		t.Logf("   * %s x%d 单价:%.2f", item.Product.Name, item.Quantity, float64(item.UnitPrice)/100)
	}
}

// 练习 1: 完善订单查询
// GetUserOrders 查询指定用户的所有订单，包含订单项和商品信息
func GetUserOrders(db *gorm.DB, userID uint) ([]Order, error) {
	var orders []Order

	err := db.
		Preload("Items.Product"). // 预加载订单项和关联的商品信息
		Where("user_id = ?", userID).
		Order("created_at DESC"). // 按创建时间降序排序
		Find(&orders).Error

	return orders, err
}

// 完善订单查询-测试函数
func TestGetUserOrders(t *testing.T) {
	db := testutil.NewTestDB(t, "ecommerce.db")

	// 确保数据存在
	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seedData(db); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	// 查询用户订单
	orders, err := GetUserOrders(db, 1)
	if err != nil {
		t.Fatalf("GetUserOrders failed: %v", err)
	}

	t.Logf("用户 1 的订单数量: %d", len(orders))
	for _, order := range orders {
		t.Logf("订单 %s: 状态=%s, 商品数量=%d",
			order.OrderNo, order.Status, len(order.Items))
		for _, item := range order.Items {
			t.Logf("  - %s x%d", item.Product.Name, item.Quantity)
		}
	}
}

// 实现库存预警
// LowStockAlert 库存预警结构体
type LowStockAlert struct {
	Product
	AlertLevel string `json:"alert_level"` // 预警级别: LOW/MEDIUM/HIGH
}

// GetLowStockProducts 查询库存低于指定阈值的商品
func GetLowStockProducts(db *gorm.DB, threshold int) ([]LowStockAlert, error) {
	var products []Product
	var alerts []LowStockAlert

	// 查询库存低于阈值的商品
	err := db.
		Where("stock < ?", threshold).
		Order("stock ASC"). // 按库存升序排序，库存最少的排前面
		Find(&products).Error

	if err != nil {
		return nil, err
	}

	// 根据库存水平设置预警级别
	for _, product := range products {
		alert := LowStockAlert{
			Product: product,
		}

		// 根据库存比例设置预警级别
		stockRatio := float64(product.Stock) / float64(threshold)
		switch {
		case stockRatio < 0.2:
			alert.AlertLevel = "HIGH"
		case stockRatio < 0.5:
			alert.AlertLevel = "MEDIUM"
		default:
			alert.AlertLevel = "LOW"
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// 扩展：获取不同预警级别的商品
func GetLowStockProductsByLevel(db *gorm.DB, threshold int, level string) ([]Product, error) {
	var products []Product

	query := db.Where("stock < ?", threshold)

	// 根据预警级别进一步过滤
	switch level {
	case "HIGH":
		query = query.Where("stock < ?", threshold/10) // 库存低于阈值的10%
	case "MEDIUM":
		query = query.Where("stock >= ? AND stock < ?", threshold/10, threshold/5)
	case "LOW":
		query = query.Where("stock >= ?", threshold/2)
	}

	err := query.Order("stock ASC").Find(&products).Error
	return products, err
}

// 实现库存预警-测试函数
func TestLowStockAlert(t *testing.T) {
	db := testutil.NewTestDB(t, "ecommerce.db")

	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seedData(db); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	// 设置库存阈值为20
	alerts, err := GetLowStockProducts(db, 20)
	if err != nil {
		t.Fatalf("GetLowStockProducts failed: %v", err)
	}

	t.Logf("库存预警商品数量: %d", len(alerts))
	for _, alert := range alerts {
		t.Logf("预警商品: %s 库存:%d 级别:%s",
			alert.Name, alert.Stock, alert.AlertLevel)
	}
}

// 扩展销售报表
// ProductSalesSummary 商品销售统计结构体
type ProductSalesSummary struct {
	ProductID   uint   `json:"product_id"`
	ProductName string `json:"product_name"`
	Sku         string `json:"sku"`
	SalesCount  int64  `json:"sales_count"`  // 销售数量
	TotalAmount int64  `json:"total_amount"` // 销售总额（分）
	AvgPrice    int64  `json:"avg_price"`    // 平均售价（分）
}

// SalesReportByProduct 按商品统计销售数据
func SalesReportByProduct(db *gorm.DB, startDate, endDate time.Time) ([]ProductSalesSummary, error) {
	var results []ProductSalesSummary

	// 构建基础查询
	query := db.Table("order_items").
		Select(`
            products.id as product_id,
            products.name as product_name,
            products.sku as sku,
            SUM(order_items.quantity) as sales_count,
            SUM(order_items.quantity * order_items.unit_price) as total_amount,
            AVG(order_items.unit_price) as avg_price
        `).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Joins("JOIN products ON products.id = order_items.product_id").
		Where("orders.status = ?", "PAID")

	// 添加时间范围过滤
	if !startDate.IsZero() {
		query = query.Where("orders.created_at >= ?", startDate.Format("2006-01-02"))
	}
	if !endDate.IsZero() {
		query = query.Where("orders.created_at <= ?", endDate.Format("2006-01-02 23:59:59"))
	}

	// 分组和排序
	err := query.
		Group("products.id, products.name, products.sku").
		Order("sales_count DESC").
		Scan(&results).Error

	return results, err
}

// 扩展原有销售报表，支持时间范围
func SalesReportWithDateRange(db *gorm.DB, startDate, endDate time.Time) ([]SalesSummary, error) {
	var rows []SalesSummary

	query := db.Table("orders").
		Select(`
            strftime('%Y-%m-%d', orders.created_at) AS day,
            COUNT(DISTINCT orders.id) AS order_count,
            SUM(order_items.quantity) AS item_count,
            SUM(order_items.quantity * order_items.unit_price) AS total_amount
        `).
		Joins("JOIN order_items ON order_items.order_id = orders.id").
		Where("orders.status = ?", "PAID")

	// 添加时间范围过滤
	if !startDate.IsZero() {
		query = query.Where("orders.created_at >= ?", startDate.Format("2006-01-02"))
	}
	if !endDate.IsZero() {
		query = query.Where("orders.created_at <= ?", endDate.Format("2006-01-02 23:59:59"))
	}

	err := query.
		Group("day").
		Order("day ASC").
		Scan(&rows).Error

	return rows, err
}

// 扩展销售报表-测试函数
func TestSalesReportByProduct(t *testing.T) {
	db := testutil.NewTestDB(t, "ecommerce.db")

	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seedData(db); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	// 查询商品销售统计
	report, err := SalesReportByProduct(db, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("SalesReportByProduct failed: %v", err)
	}

	t.Logf("商品销售统计数量: %d", len(report))
	for _, item := range report {
		t.Logf("商品: %s 销售数量:%d 销售额:%.2f 均价:%.2f",
			item.ProductName, item.SalesCount,
			float64(item.TotalAmount)/100, float64(item.AvgPrice)/100)
	}
}

// 实现订单取消
// CancelOrder 取消订单并恢复库存
func CancelOrder(ctx context.Context, db *gorm.DB, orderNo string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 锁定并加载订单及订单项
		var order Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Items").
			Where("order_no = ?", orderNo).
			First(&order).Error; err != nil {
			return fmt.Errorf("load order: %w", err)
		}

		// 2. 检查订单状态（只能取消待支付订单）
		if order.Status != "PENDING" {
			return fmt.Errorf("只能取消待支付订单，当前状态: %s", order.Status)
		}

		// 3. 恢复库存
		for _, item := range order.Items {
			// 使用原子操作增加库存
			if err := tx.Model(&Product{}).
				Where("id = ?", item.ProductID).
				UpdateColumn("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
				return fmt.Errorf("restore stock for product %d: %w", item.ProductID, err)
			}
		}

		// 4. 更新订单状态为 CANCELLED
		if err := tx.Model(&order).
			Updates(map[string]any{
				"status": "CANCELLED",
			}).Error; err != nil {
			return fmt.Errorf("update order status: %w", err)
		}

		return nil
	})
}

// 实现订单取消-测试函数
func TestCancelOrder(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t, "ecommerce.db")

	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seedData(db); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	// 先创建一个待支付订单
	products := fetchProducts(t, db)
	order, err := CreateOrder(ctx, db, 1, []OrderItemInput{
		{ProductID: products[0].ID, Quantity: 1},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	t.Logf("创建订单成功: %s", order.OrderNo)

	// 验证库存已扣减
	beforeCancel := fetchProducts(t, db)
	t.Logf("取消前库存: %s = %d", products[0].Name, beforeCancel[0].Stock)

	// 取消订单
	err = CancelOrder(ctx, db, order.OrderNo)
	if err != nil {
		t.Fatalf("CancelOrder failed: %v", err)
	}

	t.Logf("订单取消成功")

	// 验证库存已恢复
	afterCancel := fetchProducts(t, db)
	t.Logf("取消后库存: %s = %d", products[0].Name, afterCancel[0].Stock)

	// 验证订单状态
	var cancelledOrder Order
	if err := db.Where("order_no = ?", order.OrderNo).First(&cancelledOrder).Error; err != nil {
		t.Fatalf("query cancelled order failed: %v", err)
	}

	if cancelledOrder.Status != "CANCELLED" {
		t.Errorf("订单状态应为 CANCELLED，实际为: %s", cancelledOrder.Status)
	}
}
