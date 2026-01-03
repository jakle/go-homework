package basics

import (
	"fmt"
	"gohomeworklesson02/testutil"
	"testing"
	"time"

	"gorm.io/gorm"
)

// TestQueryBuilderDemo demonstrates advanced query building with GORM
// This test covers: Where conditions, Select, Order, Limit/Offset, Scopes, and aggregation queries
// Key concept: Chainable methods return *gorm.DB, allowing flexible query composition
func TestQueryBuilderDemo(t *testing.T) {
	db := testutil.NewTestDB(t, "query.db")

	type User struct {
		ID        uint
		Name      string
		Email     string
		Age       uint8
		Status    string
		CreatedAt time.Time // For ordering by creation time
	}

	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// Seed test data
	data := []User{
		{Name: "Alice", Email: "alice@example.com", Age: 28, Status: "active"},
		{Name: "Bob", Email: "bob@example.com", Age: 35, Status: "pending"},
		{Name: "Celine", Email: "celine@example.com", Age: 25, Status: "active"},
		{Name: "David", Email: "david@example.com", Age: 31, Status: "inactive"},
	}
	if err := db.Create(&data).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	// SCOPES: Reusable query conditions
	// Scopes allow you to extract common query logic into reusable functions
	// The order of chained methods doesn't matter - GORM will build the SQL correctly
	// Example: Using paginate scope for pagination
	paged := []User{}
	// Scopes: Apply reusable query conditions (paginate function)
	// Where: Add condition to filter records
	// Order: Sort results (desc = descending, asc = ascending)
	// Find: Execute query and populate the slice
	if err := db.Scopes(paginate(1, 2)).Where("status = ?", "active").Order("created_at desc").Find(&paged).Error; err != nil {
		t.Fatalf("paged query: %v", err)
	}
	if len(paged) != 2 {
		t.Fatalf("expected 2 active users, got %d", len(paged))
	}

	// WHERE with LIKE and SELECT: Specify fields to retrieve
	// Where: Use LIKE for pattern matching (SQL wildcard: %)
	// Select: Only retrieve specified fields (improves performance, reduces data transfer)
	// Note: Select can also be used with Update to specify which fields to update
	emailUsers := []User{}
	if err := db.Where("email LIKE ?", "a%").Select("id", "name", "email").Find(&emailUsers).Error; err != nil {
		t.Fatalf("email query: %v", err)
	}
	fmt.Print(emailUsers)
	if len(emailUsers) != 1 {
		t.Fatalf("expected 1 email match, got %d", len(emailUsers))
	}

	// AGGREGATION QUERIES: Group by and aggregate functions
	// Model: Specify the model for the query
	// Select: Use SQL functions like COUNT, SUM, AVG, etc.
	// Group: Group results by specified column
	// Order: Sort aggregated results
	// Scan: Use Scan instead of Find when querying into a different struct or using aggregation
	// Note: Find is for querying into the same model, Scan is for custom structures or aggregations
	type StatusCount struct {
		Status string
		Total  int64
	}
	counts := []StatusCount{}
	if err := db.Model(&User{}).Select("status, COUNT(*) as total").Group("status").Order("total desc").Scan(&counts).Error; err != nil {
		t.Fatalf("scan counts: %v", err)
	}
	fmt.Print(&counts)
	if len(counts) == 0 {
		t.Fatalf("expected counts result")
	}

	// // MULTIPLE SCOPES: Chain multiple reusable conditions
	// // Scopes can be chained together to build complex queries
	// // Each scope function receives *gorm.DB and returns *gorm.DB, enabling composition
	// // This demonstrates the power of GORM's chainable API
	filtered := []User{}
	// Multiple scopes: activeUsers() + ageBetween() + paginate()
	// Select: Specify which fields to retrieve
	// Scan: Use Scan when selecting specific fields into a struct
	if err := db.Model(&User{}).Scopes(activeUsers(), ageBetween(20, 32), paginate(1, 3)).Select("id, name, email, age").Scan(&filtered).Error; err != nil {
		t.Fatalf("filtered query: %v", err)
	}
	if len(filtered) == 0 {
		t.Fatalf("expected filtered users")
	}
}

// TestRawSQLDemo demonstrates how to mix chainable queries with GORM's native SQL APIs.
// It covers db.Raw + Scan for SELECT queries and db.Exec for UPDATE statements.
func TestRawSQLDemo(t *testing.T) {
	db := testutil.NewTestDB(t, "raw.db")

	type User struct {
		ID          uint
		Name        string
		Age         uint8
		Status      string
		CreatedAt   time.Time
		LastLoginAt time.Time
	}

	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	now := time.Now()
	data := []User{
		{Name: "Alice", Age: 28, Status: "active", CreatedAt: now.AddDate(0, 0, -10), LastLoginAt: now.AddDate(0, 0, -5)},
		{Name: "Bob", Age: 35, Status: "pending", CreatedAt: now.AddDate(0, 0, -20), LastLoginAt: now.AddDate(0, 0, -40)},
		{Name: "Celine", Age: 25, Status: "active", CreatedAt: now.AddDate(0, 0, -15), LastLoginAt: now.AddDate(0, 0, -60)},
	}
	if err := db.Create(&data).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	// Raw query with aggregation
	type StatusSummary struct {
		Status string
		Total  int64
		AvgAge float64
	}
	stats := []StatusSummary{}
	start := now.AddDate(0, 0, -60)
	end := now
	if err := db.Raw(`
        SELECT status, COUNT(*) AS total, AVG(age) AS avg_age
        FROM users
        WHERE created_at BETWEEN ? AND ?
        GROUP BY status
    `, start, end).Scan(&stats).Error; err != nil {
		t.Fatalf("raw scan: %v", err)
	}
	if len(stats) == 0 {
		t.Fatalf("expected status summaries, got 0")
	}
	fmt.Println(stats)

	// Exec statement for bulk updates
	threshold := now.AddDate(0, 0, -30)
	result := db.Exec("UPDATE users SET status = ? WHERE last_login_at < ?", "inactive", threshold)
	if result.Error != nil {
		t.Fatalf("exec update: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		t.Fatalf("expected affected rows")
	}

	var inactiveCount int64
	if err := db.Model(&User{}).Where("status = ?", "inactive").Count(&inactiveCount).Error; err != nil {
		t.Fatalf("count inactive: %v", err)
	}
	if inactiveCount == 0 {
		t.Fatalf("expected inactive users after exec")
	}
}

// paginate is a Scope function that implements pagination
// Scope functions: Return a function that takes *gorm.DB and returns *gorm.DB
// This pattern allows reusable query conditions to be applied via db.Scopes()
// Usage: db.Scopes(paginate(1, 10)).Find(&users)
// - page: Page number (1-indexed)
// - size: Number of records per page
// Returns: A scope function that applies Offset and Limit
func paginate(page, size int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// Validate and normalize page number
		if page <= 0 {
			page = 1
		}
		// Validate and normalize page size (max 100, min 10)
		switch {
		case size > 100:
			size = 100
		case size <= 0:
			size = 10
		}
		// Calculate offset: (page - 1) * size
		// Example: page 1, size 10 -> offset 0
		//          page 2, size 10 -> offset 10
		offset := (page - 1) * size
		// Offset: Skip N records
		// Limit: Return at most N records
		return db.Offset(offset).Limit(size)
	}
}

// activeUsers is a Scope function that filters for active users
// This demonstrates extracting common query conditions into reusable scopes
// Usage: db.Scopes(activeUsers()).Find(&users)
func activeUsers() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", "active")
	}
}

// ageBetween is a Scope function that filters users by age range
// This demonstrates parameterized scopes for flexible query building
// Usage: db.Scopes(ageBetween(20, 30)).Find(&users)
// - min: Minimum age (inclusive)
// - max: Maximum age (inclusive)
func ageBetween(min, max int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// BETWEEN: SQL operator for range queries
		// Equivalent to: age >= min AND age <= max
		return db.Where("age BETWEEN ? AND ?", min, max)
	}
}
