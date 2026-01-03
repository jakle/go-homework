package basics

import (
	"fmt"
	"gohomeworklesson02/testutil"
	"testing"
	"time"

	"gorm.io/gorm"
)

type User1 struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	Email       string `gorm:"uniqueIndex"`
	Age         uint8
	Status      string
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// 定义全局常量
const (
	MinAge = 18
	MaxAge = 30
)

// YoungUsers 创建一个查询年龄在 18-30 岁之间用户的 scope
// 这个函数返回一个闭包，闭包接收 *gorm.DB 并返回修改后的 *gorm.DB
// 用法: db.Scopes(YoungUsers()).Find(&users)
func YoungUsers() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("age >= ? AND age <= ?", MinAge, MaxAge)
	}
}

// YoungUsersWithStatus 创建一个查询年龄在 18-30 岁之间且具有特定状态的用户的 scope
// 支持链式调用: db.Scopes(YoungUsersWithStatus("active")).Find(&users)
func YoungUsersWithStatus(status string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("age >= ? AND age <= ? AND status = ?", MinAge, MaxAge, status)
	}
}

// ActiveYoungUsers 创建一个查询年龄在 18-30 岁之间且状态为 active 的用户的 scope
func ActiveYoungUsers() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("age >= ? AND age <= ? AND status = ?", MinAge, MaxAge, "active")
	}
}

// YoungUsersOrdered 创建一个查询年龄在 18-30 岁之间并排序的 scope
// 参数 orderBy: 排序字段，如 "age", "created_at"
// 参数 order: 排序方式，"asc" 或 "desc"
func YoungUsersOrdered(orderBy, order string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if orderBy == "" {
			orderBy = "age"
		}
		if order == "" {
			order = "asc"
		}
		tx := db.Where("age >= ? AND age <= ?", MinAge, MaxAge).
			Order(fmt.Sprintf("%s %s", orderBy, order))
		return tx
	}
}

// 分页相关的 scopes

// Paginate 通用分页 scope
// 参数 page: 页码（从1开始）
// 参数 size: 每页大小
func Paginate(page, size int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page < 1 {
			page = 1
		}
		if size <= 0 {
			size = 20
		}
		if size > 100 {
			size = 100
		}
		offset := (page - 1) * size
		return db.Offset(offset).Limit(size)
	}
}

// GetYoungUsersWithPagination 使用 scope 查询年轻用户（分页版本）
func GetYoungUsersWithPagination(db *gorm.DB, page, size int) ([]User1, int64, error) {
	var users []User1
	var total int64

	// 先获取总数
	if err := db.Model(&User1{}).Scopes(YoungUsers()).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取年轻用户总数失败: %w", err)
	}

	// 再获取分页数据
	if err := db.Scopes(
		YoungUsers(),
		Paginate(page, size),
	).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("分页查询年轻用户失败: %w", err)
	}

	return users, total, nil
}

// GetYoungUsersByPage 多条件查询年轻用户
func GetYoungUsersByPage(db *gorm.DB, page, size int, status, orderBy, order string) ([]User1, int64, error) {
	var users []User1
	var total int64

	// 构建基础查询
	query := db.Model(&User1{}).Where("age >= ? AND age <= ?", MinAge, MaxAge)

	// 添加状态筛选
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取总数失败: %w", err)
	}

	// 添加排序
	if orderBy != "" {
		if order == "" {
			order = "asc"
		}
		query = query.Order(fmt.Sprintf("%s %s", orderBy, order))
	} else {
		query = query.Order("created_at DESC")
	}

	// 添加分页
	offset := (page - 1) * size
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	// 执行查询
	if err := query.Offset(offset).Limit(size).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("查询年轻用户失败: %w", err)
	}

	return users, total, nil
}

// 使用 scopes 的高级查询示例

// FindYoungUsersByEmail 按邮箱模糊查询年轻用户
func FindYoungUsersByEmail(db *gorm.DB, emailPattern string, page, size int) ([]User1, int64, error) {
	var users []User1
	var total int64

	// 使用链式调用和 scopes
	baseQuery := db.Model(&User1{}).Scopes(YoungUsers())

	// 添加邮箱筛选
	if emailPattern != "" {
		baseQuery = baseQuery.Where("email LIKE ?", "%"+emailPattern+"%")
	}

	// 获取总数
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取总数失败: %w", err)
	}

	// 分页查询
	if err := baseQuery.Scopes(
		Paginate(page, size),
	).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("查询失败: %w", err)
	}

	return users, total, nil
}

// 测试函数
func TestScopes(t *testing.T) {
	db := testutil.NewTestDB(t, "scopes_test.db")

	// 自动迁移
	if err := db.AutoMigrate(&User1{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 创建测试数据
	seedUsers := []User1{
		{Name: "Alice", Email: "alice@example.com", Age: 25, Status: "active"},
		{Name: "Bob", Email: "bob@example.com", Age: 17, Status: "active"},         // 小于18岁
		{Name: "Charlie", Email: "charlie@example.com", Age: 30, Status: "active"}, // 刚好30岁
		{Name: "David", Email: "david@example.com", Age: 31, Status: "active"},     // 大于30岁
		{Name: "Eve", Email: "eve@example.com", Age: 22, Status: "inactive"},
		{Name: "Frank", Email: "frank@example.com", Age: 28, Status: "active"},
		{Name: "Grace", Email: "grace@example.com", Age: 20, Status: "active"},
		{Name: "Henry", Email: "henry@example.com", Age: 18, Status: "active"}, // 刚好18岁
		{Name: "Ivy", Email: "ivy@example.com", Age: 29, Status: "active"},
		{Name: "Jack", Email: "jack@example.com", Age: 35, Status: "active"},
		{Name: "Kate", Email: "kate@example.com", Age: 19, Status: "suspended"},
		{Name: "Leo", Email: "leo@example.com", Age: 26, Status: "active"},
	}

	for i := range seedUsers {
		if err := db.Create(&seedUsers[i]).Error; err != nil {
			t.Fatalf("创建用户失败: %v", err)
		}
	}

	t.Run("测试分页查询", func(t *testing.T) {
		// 测试 GetYoungUsersWithPagination
		users, total, err := GetYoungUsersWithPagination(db, 1, 3)
		if err != nil {
			t.Fatalf("分页查询失败: %v", err)
		}

		if total != 9 {
			t.Errorf("预期总数 9，实际 %d", total)
		}

		if len(users) != 3 {
			t.Errorf("预期第1页3条记录，实际 %d 条", len(users))
		}

		t.Logf("总数: %d, 第1页: %d 条记录", total, len(users))

		// 测试第2页
		users2, _, err := GetYoungUsersWithPagination(db, 2, 3)
		if err != nil {
			t.Fatalf("分页查询失败: %v", err)
		}

		t.Logf("第2页: %d 条记录", len(users2))
	})

	t.Run("测试排序 scope", func(t *testing.T) {
		// 测试按年龄升序
		var usersAsc []User1
		if err := db.Scopes(
			YoungUsers(),
			YoungUsersOrdered("age", "asc"),
		).Find(&usersAsc).Error; err != nil {
			t.Fatalf("查询排序用户失败: %v", err)
		}

		// 验证升序
		for i := 1; i < len(usersAsc); i++ {
			if usersAsc[i-1].Age > usersAsc[i].Age {
				t.Errorf("年龄不是升序: [%d]%d > [%d]%d",
					i-1, usersAsc[i-1].Age, i, usersAsc[i].Age)
			}
		}

		t.Logf("按年龄升序: 从 %d 岁到 %d 岁",
			usersAsc[0].Age, usersAsc[len(usersAsc)-1].Age)

		// 测试按年龄降序
		var usersDesc []User1
		if err := db.Scopes(
			YoungUsers(),
			YoungUsersOrdered("age", "desc"),
		).Find(&usersDesc).Error; err != nil {
			t.Fatalf("查询排序用户失败: %v", err)
		}

		// 验证降序
		for i := 1; i < len(usersDesc); i++ {
			if usersDesc[i-1].Age < usersDesc[i].Age {
				t.Errorf("年龄不是降序: [%d]%d < [%d]%d",
					i-1, usersDesc[i-1].Age, i, usersDesc[i].Age)
			}
		}

		t.Logf("按年龄降序: 从 %d 岁到 %d 岁",
			usersDesc[0].Age, usersDesc[len(usersDesc)-1].Age)
	})

	t.Run("测试多 scope 组合", func(t *testing.T) {
		// 测试多个 scopes 的组合使用
		var users []User1
		if err := db.Scopes(
			YoungUsers(),                   // 筛选年龄
			YoungUsersWithStatus("active"), // 筛选状态
			Paginate(1, 2),                 // 分页
		).Order("age ASC").Find(&users).Error; err != nil {
			t.Fatalf("组合查询失败: %v", err)
		}

		if len(users) != 2 {
			t.Errorf("预期2条记录，实际 %d 条", len(users))
		}

		// 验证结果
		for _, user := range users {
			if user.Age < 18 || user.Age > 30 {
				t.Errorf("用户 %s 年龄 %d 不在 18-30 范围内", user.Name, user.Age)
			}
			if user.Status != "active" {
				t.Errorf("用户 %s 状态不是 active，实际是 %s", user.Name, user.Status)
			}
		}

		t.Logf("组合查询结果: %d 条记录", len(users))
	})

	t.Run("测试 FindYoungUsersByEmail", func(t *testing.T) {
		// 测试邮箱模糊查询
		users, total, err := FindYoungUsersByEmail(db, "example", 1, 5)
		if err != nil {
			t.Fatalf("邮箱模糊查询失败: %v", err)
		}

		if total < 5 {
			t.Logf("注意: 查询结果少于5条")
		}

		t.Logf("邮箱包含 'example' 的年轻用户: 总数 %d, 本页 %d 条", total, len(users))

		// 测试特定邮箱
		users, total, err = FindYoungUsersByEmail(db, "alice", 1, 10)
		if err != nil {
			t.Fatalf("邮箱模糊查询失败: %v", err)
		}

		if total != 1 {
			t.Errorf("预期找到1个包含 'alice' 的用户，实际 %d 个", total)
		} else if users[0].Name != "Alice" {
			t.Errorf("预期找到 Alice，实际找到 %s", users[0].Name)
		}

		t.Logf("邮箱包含 'alice' 的用户: %s", users[0].Name)
	})

}
