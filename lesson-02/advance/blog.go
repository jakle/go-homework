package main

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"time"
)

type User struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	Email     string `gorm:"uniqueIndex"`
	Posts     []Post `gorm:"foreignKey:UserID"`
	PostCount uint   `gorm:"default:0"` // 用于统计用户文章数量
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Post struct {
	ID        uint `gorm:"primaryKey"`
	Title     string
	Content   string
	UserID    uint      // Belongs To User
	User      User      `gorm:"foreignKey:UserID"`
	Comments  []Comment `gorm:"foreignKey:PostID"`
	Tags      []Tag     `gorm:"many2many:post_tags;"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"` // 软删除
}

type Comment struct {
	ID        uint `gorm:"primaryKey"`
	Content   string
	UserID    uint
	PostID    uint
	Post      Post `gorm:"foreignKey:PostID"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"` // 软删除
}

type Tag struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"uniqueIndex"`
	Posts     []Post `gorm:"many2many:post_tags;"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PostWithCount 用于包含评论数量的文章
type PostWithCount struct {
	Post
	CommentCount int64 `json:"comment_count"`
}

// 查询用户最新文章（含标签）
func GetUserLatestPosts(db *gorm.DB, userID uint) ([]Post, error) {
	var posts []Post

	err := db.
		Model(&Post{}).
		Where("user_id = ?", userID).
		Preload("User").
		Preload("Tags").
		Preload("Comments").
		Order("created_at DESC").
		Limit(10).
		Find(&posts).Error

	return posts, err
}

// 统计评论数量
func GetPostsWithCommentCount(db *gorm.DB) ([]PostWithCount, error) {
	var posts []Post
	var result []PostWithCount

	// 先查询文章并预加载评论
	err := db.
		Model(&Post{}).
		Preload("Comments").
		Preload("User").
		Preload("Tags").
		Find(&posts).Error

	if err != nil {
		return nil, err
	}

	// 转换结果，包含评论数量
	for _, post := range posts {
		var count int64

		// 统计评论数量
		err := db.Model(&Comment{}).
			Where("post_id = ?", post.ID).
			Count(&count).Error

		if err != nil {
			return nil, err
		}

		result = append(result, PostWithCount{
			Post:         post,
			CommentCount: count,
		})
	}

	return result, nil
}

// 发布文章并绑定标签
func PublishPostWithTags(db *gorm.DB, post *Post, tagIDs []uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 1. 创建文章
		if err := tx.Create(post).Error; err != nil {
			return err
		}

		// 2. 绑定标签（Many to Many）
		if len(tagIDs) > 0 {
			// 查询标签
			var tags []Tag
			if err := tx.Where("id IN ?", tagIDs).Find(&tags).Error; err != nil {
				return err
			}

			// 关联标签
			if err := tx.Model(post).Association("Tags").Append(&tags); err != nil {
				return err
			}
		}

		// 3. 更新用户文章数量
		if err := tx.Model(&User{}).
			Where("id = ?", post.UserID).
			UpdateColumn("post_count", gorm.Expr("post_count + ?", 1)).
			Error; err != nil {
			return err
		}

		return nil
	})
}

// 发布评论函数
func PublishComment(db *gorm.DB, userID, postID uint, content string) (*Comment, error) {
	comment := &Comment{
		Content:   content,
		UserID:    userID,
		PostID:    postID,
		CreatedAt: time.Now(),
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		// 验证用户和文章是否存在
		var userCount, postCount int64
		if err := tx.Model(&User{}).Where("id = ?", userID).Count(&userCount).Error; err != nil {
			return err
		}
		if userCount == 0 {
			return fmt.Errorf("用户不存在")
		}

		if err := tx.Model(&Post{}).Where("id = ?", postID).Count(&postCount).Error; err != nil {
			return err
		}
		if postCount == 0 {
			return fmt.Errorf("文章不存在")
		}

		// 创建评论
		if err := tx.Create(comment).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 预加载关联数据
	db.Preload("User").Preload("Post").First(comment, comment.ID)
	return comment, nil
}

// 获取文章的所有评论（包含用户信息）
func GetPostComments(db *gorm.DB, postID uint) ([]Comment, error) {
	var comments []Comment

	err := db.
		Model(&Comment{}).
		Where("post_id = ?", postID).
		Preload("User").         // 预加载用户信息
		Order("created_at ASC"). // 按时间正序排列
		Find(&comments).Error

	return comments, err
}

// 获取用户的评论历史
func GetUserComments(db *gorm.DB, userID uint) ([]Comment, error) {
	var comments []Comment

	err := db.
		Model(&Comment{}).
		Where("user_id = ?", userID).
		Preload("Post").          // 预加载文章信息
		Preload("User").          // 预加载用户信息
		Order("created_at DESC"). // 按时间倒序排列
		Find(&comments).Error

	return comments, err
}

// 软删除评论
func SoftDeleteComment(db *gorm.DB, commentID uint) error {
	return db.Delete(&Comment{}, commentID).Error
}

// 彻底删除评论
func HardDeleteComment(db *gorm.DB, commentID uint) error {
	return db.Unscoped().Delete(&Comment{}, commentID).Error
}

func main() {
	// 连接数据库
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// 自动迁移
	err = db.AutoMigrate(&User{}, &Post{}, &Comment{}, &Tag{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("数据库连接成功！")

	// 示例：创建用户
	user := User{
		Name:  "张三",
		Email: "zhangsan@example.com",
	}
	db.Create(&user)

	// 发布文章
	post := &Post{
		Title:   "GORM教程",
		Content: "这是一篇关于GORM的教程",
		UserID:  user.ID,
	}

	tagIDs := []uint{1, 2, 3}
	err = PublishPostWithTags(db, post, tagIDs)
	if err != nil {
		log.Printf("发布文章失败: %v", err)
	}

	// 查询用户最新文章
	latestPosts, err := GetUserLatestPosts(db, user.ID)
	if err != nil {
		log.Printf("查询用户最新文章失败: %v", err)
	} else {
		fmt.Printf("用户 %s 的最新文章: %d 篇\n", user.Name, len(latestPosts))
	}

	comment1, err := PublishComment(db, user.ID, post.ID, "这篇博客写得真不错！")
	if err != nil {
		fmt.Printf("发布评论失败: %v\n", err)
	} else {
		fmt.Printf("用户 %s 评论: %s\n", user.Name, comment1.Content)
	}

	// 示例：软删除评论
	var comment Comment
	if err := db.First(&comment).Error; err == nil {
		err = SoftDeleteComment(db, comment.ID)
		if err != nil {
			log.Printf("软删除评论失败: %v", err)
		}
	}
}
