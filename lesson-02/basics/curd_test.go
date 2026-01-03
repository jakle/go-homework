package basics

import (
	"errors"
	"fmt"
	"gohomeworklesson02/testutil"
	"testing"
	"time"

	"gorm.io/gorm"
)

// TestCRUDDemo demonstrates the complete CRUD operations in GORM
// This test covers Create, Read, Update, and Delete operations with various patterns
func TestCRUDDemo(t *testing.T) {
	db := testutil.NewTestDB(t, "crud.db")

	// Define the User model
	// GORM will automatically map this struct to a "users" table
	type User struct {
		ID          uint       `gorm:"primaryKey"` // Primary key, auto-increment
		Name        string     // Regular field
		Email       string     `gorm:"uniqueIndex"`         // Unique index for email
		Phone       string     `gorm:"uniqueIndex;size:20"` // 新增：电话号码字段，唯一索引，指定长度优化存储
		Age         uint8      // Age field
		Status      string     // Status field
		LastLoginAt *time.Time // 新增：最后登录时间字段，使用指针类型，允许NULL
		CreatedAt   time.Time  // GORM will auto-populate on create
		UpdatedAt   time.Time  // GORM will auto-populate on create/update
	}

	// AutoMigrate creates the table if it doesn't exist
	// It will also add new columns if the struct has new fields
	// Note: It will NOT delete existing columns or modify existing data
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	now := time.Now()

	// Seed initial data: batch insert using Create
	// Create can accept a single struct or a slice for batch insertion
	seed := []User{
		{Name: "Alice", Email: "alice@example.com", Age: 28, Status: "active", LastLoginAt: &now, Phone: "12345678901"},
		{Name: "Alice1", Email: "alice1@example.com", Age: 28, Status: "active", LastLoginAt: &now, Phone: "12345678902"},
		{Name: "Alice2", Email: "alice2@example.com", Age: 28, Status: "inactive", LastLoginAt: &now, Phone: "12345678903"},
		{Name: "Alice3", Email: "alice3@example.com", Age: 28, Status: "inactive", LastLoginAt: &now, Phone: "12345678904"},
		{Name: "Bob", Email: "bob@example.com", Age: 31, Status: "active", LastLoginAt: &now, Phone: "12345678905"},
		{Name: "Bob1", Email: "bob1@example.com", Age: 32, Status: "active", LastLoginAt: &now, Phone: "12345678906"},
		{Name: "Bob2", Email: "bob2@example.com", Age: 33, Status: "inactive", LastLoginAt: &now, Phone: "12345678907"},
		{Name: "Bob3", Email: "bob3example.com", Age: 34, Status: "inactive", LastLoginAt: &now, Phone: "12345678908"},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	// CREATE: Single record insertion
	// Create inserts a new record and automatically populates:
	// - Primary key (ID) after insertion
	// - CreatedAt and UpdatedAt timestamps
	// You can also use Select/Omit to control which fields are inserted
	t.Run("create", func(t *testing.T) {
		u := User{Name: "Diane", Email: "diane@example.com", Age: 30, Status: "active", Phone: "12345678909", LastLoginAt: &now}
		// Create returns the inserted record with ID populated
		if err := db.Create(&u).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
		// After Create, u.ID is automatically populated by GORM
		t.Logf("new user id=%d", u.ID)
	})

	// 测试新增字段的CRUD操作
	t.Run("phone op", func(t *testing.T) {
		// 测试通过Phone查询
		var userByPhone User
		if err := db.Where("phone = ?", "12345678909").First(&userByPhone).Error; err != nil {
			t.Fatalf("query user by phone: %v", err)
		}
		t.Logf("user by phone 12345678909: %s, email: %s", userByPhone.Name, userByPhone.Email)

		// 测试Phone唯一性约束
		duplicateUser := User{
			Name:  "Duplicate",
			Email: "duplicate@example.com",
			Phone: "12345678909", // 重复的手机号
			Age:   25,
		}
		if err := db.Create(&duplicateUser).Error; err == nil {
			t.Fatal("expected error for duplicate phone, but got none")
		} else {
			t.Logf("correctly rejected duplicate phone: %v", err)
		}
	})

	t.Run("last login op", func(t *testing.T) {
		// 测试查询有最后登录时间的用户
		var recentUsers []User
		if err := db.Where("last_login_at IS NOT NULL").Order("last_login_at desc").Find(&recentUsers).Error; err != nil {
			t.Fatalf("query users with last login: %v", err)
		}
		t.Logf("users with last login: %d", len(recentUsers))
		for _, u := range recentUsers {
			t.Logf("  - %s: last login at %v", u.Name, u.LastLoginAt)
		}

		// 测试更新最后登录时间
		newLoginTime := time.Now()
		var userToUpdate User
		if err := db.Where("email = ?", "alice1@example.com").First(&userToUpdate).Error; err != nil {
			t.Fatalf("find user: %v", err)
		}

		// 更新最后登录时间
		if err := db.Model(&userToUpdate).Update("last_login_at", newLoginTime).Error; err != nil {
			t.Fatalf("update last login: %v", err)
		}

		// 验证更新
		var updatedUser User
		if err := db.First(&updatedUser, userToUpdate.ID).Error; err != nil {
			t.Fatalf("reload user: %v", err)
		}
		if updatedUser.LastLoginAt == nil || !updatedUser.LastLoginAt.Equal(newLoginTime) {
			t.Fatalf("last login not updated correctly")
		}
		t.Logf("updated user last login: %v", updatedUser.LastLoginAt)
	})

	// // READ: Query operations
	// // GORM provides several query methods:
	// // - First: Get the first record matching conditions (returns error if not found)
	// // - Take: Get one record (doesn't require conditions)
	// // - Find: Get all matching records (returns empty slice if none found)
	// // - Scan: Scan results into a struct or map
	// // Always check for gorm.ErrRecordNotFound when using First

	t.Run("query/first", func(t *testing.T) {
		var user User
		// First: Get the first record matching conditions
		// Returns gorm.ErrRecordNotFound if no record found
		// Can use conditions: db.First(&user, "email = ?", "alice@example.com")
		// Or primary key: db.First(&user, 1)
		if err := db.Where("status = ?", "active").First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("no active user found")
			}
			t.Fatalf("query first active user: %v", err)
		}
		t.Logf("first active user: %+v", user)

		// First with primary key
		var userByID User
		if err := db.First(&userByID, 1).Error; err != nil {
			t.Fatalf("query user by ID: %v", err)
		}
		t.Logf("user by ID 1: %+v", userByID)
	})

	t.Run("query/take", func(t *testing.T) {
		var user User
		// Take: Get one record without requiring conditions
		// Doesn't return error if no record found (just doesn't populate the struct)
		// Useful when you just want any record from the table
		if err := db.Take(&user).Error; err != nil {
			t.Fatalf("take user: %v", err)
		}
		t.Logf("taken user: %+v", user)

		// Take with conditions
		var activeUser User
		if err := db.Where("status = ?", "inactive").Take(&activeUser).Error; err != nil {
			t.Fatalf("take active user: %v", err)
		}
		t.Logf("taken active user: %+v", activeUser)
	})

	t.Run("query/find", func(t *testing.T) {
		var actives []User
		// Find: Get all matching records
		// Returns empty slice if no records found (no error)
		// Where: Add conditions to the query
		// Order: Sort results (asc/desc)
		if err := db.Where("status = ?", "active").Order("created_at desc").Find(&actives).Error; err != nil {
			t.Fatalf("query actives: %v", err)
		}
		if len(actives) == 0 {
			t.Fatalf("expected at least one active user")
		}
		t.Logf("active users: %+v", actives)

		// Find all records
		var allUsers []User
		if err := db.Find(&allUsers).Error; err != nil {
			t.Fatalf("find all users: %v", err)
		}
		t.Logf("all users count: %d", len(allUsers))
	})

	t.Run("query/scan", func(t *testing.T) {
		// Scan: Scan results into a struct or map
		// Useful when you only need specific fields or want to scan into a different structure
		type UserSummary struct {
			Name   string
			Email  string
			Status string
		}
		var summaries []UserSummary
		// Select specific fields and scan into a different struct
		if err := db.Model(&User{}).Select("name", "email", "status").Where("status = ?", "active").Scan(&summaries).Error; err != nil {
			t.Fatalf("scan user summaries: %v", err)
		}
		t.Logf("user summaries: %+v", summaries)

		// Scan into a map
		var result map[string]interface{}
		if err := db.Model(&User{}).Select("name", "email", "age").Where("email = ?", "alice@example.com").Scan(&result).Error; err != nil {
			t.Fatalf("scan to map: %v", err)
		}
		t.Logf("user as map: %+v", result)

		// Scan into primitive values
		var count int64
		if err := db.Model(&User{}).Where("status = ?", "active").Count(&count).Error; err != nil {
			t.Fatalf("count active users: %v", err)
		}
		t.Logf("active users count: %d", count)
	})

	// // UPDATE: Update operations
	// // GORM provides different update methods:
	// // - Save: Updates all fields (including zero values)
	// // - Updates: Updates specified fields (ignores zero values by default)
	// // - Update: Updates a single field
	// // Use Select to specify which fields to update, or Omit to exclude fields
	// // Model(&user) is used to specify the model for the update operation
	t.Run("update", func(t *testing.T) {
		var user User
		// First: Get the first record matching the condition
		// Second parameter can be a condition string or primary key value
		if err := db.First(&user, "email = ?", "diane@example.com").Error; err != nil {
			t.Errorf("load user: %v", err)
		}
		fmt.Print(&user)
		// Select: Only update specified fields (Age and Status)
		// This prevents updating other fields and ignores zero values for non-selected fields
		if err := db.Model(&user).Select("Age", "Status").Where("email = ?", "alice@example.com").Updates(User{Age: 31, Status: "vip"}).Error; err != nil {
			t.Fatalf("update fields: %v", err)
		}
		// Reload the user to verify the update
		// First with ID: Query by primary key
		if err := db.First(&user, 1).Error; err != nil {
			t.Fatalf("reload user: %v", err)
		}
		if user.Age != 31 || user.Status != "vip" {
			t.Fatalf("unexpected updated values: %+v", user)
		}
	})

	// // BULK UPDATE: Update multiple records at once
	// // Use Model(&User{}) without a specific instance to perform bulk operations
	// // Updates can accept a struct or a map[string]any
	// // RowsAffected indicates how many rows were actually updated
	t.Run("bulk update", func(t *testing.T) {
		// Model(&User{}): Specify the model for bulk operation
		// Where: Add conditions to filter which records to update
		// Updates: Update all matching records
		// Using map[string]any allows updating specific fields without zero value issues
		res := db.Model(&User{}).Where("status = ?", "inactive").Updates(map[string]any{"status": "pending_review"})
		if res.Error != nil {
			t.Fatalf("bulk update: %v", res.Error)
		}
		// RowsAffected: Check how many rows were actually updated
		if res.RowsAffected == 0 {
			t.Fatalf("expected rows to be updated")
		}
	})

	// // DELETE: Delete operations
	// // Delete can be used with:
	// // - A specific instance: db.Delete(&user)
	// // - A model with conditions: db.Delete(&User{}, "id = ?", id)
	// // - Bulk delete: db.Where(...).Delete(&User{})
	// // Note: Soft delete will be covered in the advanced section
	// // After deletion, querying the record should return gorm.ErrRecordNotFound
	t.Run("delete", func(t *testing.T) {
		var user User
		// First: Load the user to delete
		if err := db.First(&user, "email = ?", "alice1@example.com").Error; err != nil {
			t.Fatalf("load user: %v", err)
		}
		// Delete: Delete by primary key
		// First parameter is the model type, second is the primary key value
		if err := db.Delete(&User{}, user.ID).Error; err != nil {
			t.Fatalf("delete: %v", err)
		}
		// Verify deletion: Query should return gorm.ErrRecordNotFound
		// Always use errors.Is to check for gorm.ErrRecordNotFound
		err := db.First(&User{}, user.ID).Error
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})
}

// User 模型定义
type User struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	Email       string `gorm:"uniqueIndex"`
	Phone       string `gorm:"uniqueIndex;size:20"`
	Age         uint8
	Status      string
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

/*
CreateUser 新增用户：创建用户并默认开启激活状态
参数：
  - db: GORM 数据库连接
  - name: 用户名
  - email: 邮箱

返回值：
  - *User: 创建的用户对象
  - error: 错误信息
*/
func CreateUser(db *gorm.DB, name, email string) (*User, error) {
	// 参数验证
	if name == "" {
		return nil, errors.New("用户名不能为空")
	}
	if email == "" {
		return nil, errors.New("邮箱不能为空")
	}

	// 检查邮箱是否已存在
	var existingUser User
	err := db.Where("email = ?", email).First(&existingUser).Error
	if err == nil {
		// 如果找到了现有用户，返回错误
		return nil, errors.New("邮箱已被注册")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果是其他数据库错误，返回该错误
		return nil, fmt.Errorf("检查邮箱失败: %w", err)
	}

	// 创建用户实例，设置默认值
	user := &User{
		Name:   name,
		Email:  email,
		Status: "active", // 默认开启激活状态
		Age:    0,        // 默认年龄为0，可根据需求调整
	}

	// 创建用户
	if err := db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return user, nil
}

/*
SearchUsersByEmail 模糊查询：根据邮箱模糊查询用户列表（支持分页）
参数：
  - db: GORM 数据库连接
  - emailPattern: 邮箱匹配模式，如 "%example.com"、"alice%"、"%alice%"
  - page: 页码（从1开始）
  - size: 每页大小

返回值：
  - []User: 用户列表
  - error: 错误信息
*/
func SearchUsersByEmail(db *gorm.DB, emailPattern string, page, size int) ([]User, error) {
	// 参数验证
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20 // 默认每页20条
	}

	// 计算偏移量
	offset := (page - 1) * size

	var users []User

	// 构建查询
	query := db.Model(&User{}).Where("email LIKE ?", emailPattern)

	// 添加排序（默认按创建时间倒序）
	query = query.Order("created_at DESC")

	// 执行分页查询
	if err := query.Offset(offset).Limit(size).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return users, nil
}

/*
UpdateUserStatus 批量更新状态：批量更新用户状态
参数：
  - db: GORM 数据库连接
  - ids: 用户ID数组
  - status: 新的状态值

返回值：
  - error: 错误信息
*/
func UpdateUserStatus(db *gorm.DB, ids []uint, status string) error {
	// 参数验证
	if len(ids) == 0 {
		return errors.New("用户ID列表不能为空")
	}
	if status == "" {
		return errors.New("状态不能为空")
	}

	// 验证状态值的有效性
	validStatuses := []string{"active", "inactive", "pending", "suspended", "vip"}
	valid := false
	for _, s := range validStatuses {
		if s == status {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("无效的状态值: %s，有效值: %v", status, validStatuses)
	}

	// 批量更新
	result := db.Model(&User{}).Where("id IN ?", ids).Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("更新状态失败: %w", result.Error)
	}

	// 检查是否有实际更新的记录
	if result.RowsAffected == 0 {
		return errors.New("没有找到符合条件的用户")
	}

	return nil
}

/*
DeleteInactiveUsers 删除过期用户：删除超过 30 天未登录的用户
注意：这是硬删除，会从数据库中永久删除数据
在生产环境中，通常建议使用软删除
*/
func DeleteInactiveUsers(db *gorm.DB) error {
	// 计算30天前的时间
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)

	// 使用事务确保数据一致性
	err := db.Transaction(func(tx *gorm.DB) error {
		// 先查询要删除的用户信息（用于日志或其他用途）
		var usersToDelete []User
		if err := tx.Where("last_login_at IS NULL OR last_login_at < ?", thirtyDaysAgo).Find(&usersToDelete).Error; err != nil {
			return fmt.Errorf("查询过期用户失败: %w", err)
		}

		// 如果没有用户需要删除，直接返回
		if len(usersToDelete) == 0 {
			return nil
		}

		// 记录要删除的用户信息
		userIDs := make([]uint, len(usersToDelete))
		for i, user := range usersToDelete {
			userIDs[i] = user.ID
		}

		// 执行删除
		result := tx.Where("id IN ?", userIDs).Delete(&User{})
		if result.Error != nil {
			return fmt.Errorf("删除用户失败: %w", result.Error)
		}

		// 记录删除的数量
		tx.Commit()

		return nil
	})

	return err
}
