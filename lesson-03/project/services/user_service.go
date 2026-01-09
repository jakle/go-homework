package services

import (
	"errors"
	"fmt"
	"gin-examples/project/config"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"gin-examples/project/models"
	"gin-examples/project/utils"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	db     *gorm.DB
	config *config.Config
}

func NewUserService(db *gorm.DB, cfg *config.Config) *UserService {
	return &UserService{db: db, config: cfg}
}

func (s *UserService) CreateUser(req models.CreateUserRequest) (*models.User, error) {
	// 检查用户名是否已存在
	var existingUser models.User
	if err := s.db.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		return nil, utils.NewAppError(409, "Username already exists")
	}

	// 检查邮箱是否已存在
	if err := s.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return nil, utils.NewAppError(409, "Email already exists")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := models.User{
		Username: req.Username,
		Email:    req.Email,
		Role:     req.Role,
		Password: string(hashedPassword),
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *UserService) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, utils.NewAppError(404, "User not found")
		}
		return nil, err
	}
	return &user, nil
}

func (s *UserService) Authenticate(username, password string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, utils.NewAppError(401, "Invalid credentials")
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, utils.NewAppError(401, "Invalid credentials")
	}

	return &user, nil
}

func (s *UserService) UpdateUser(id uint, req models.UpdateUserRequest) (*models.User, error) {
	user, err := s.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	// 如果更新邮箱，检查是否已存在
	if req.Email != "" && req.Email != user.Email {
		var existingUser models.User
		if err := s.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
			return nil, utils.NewAppError(409, "Email already exists")
		}
		user.Email = req.Email
	}

	if err := s.db.Save(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) UpdateProfile(id uint, req models.UpdateProfileRequest) (*models.User, error) {
	user, err := s.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	// 如果更新邮箱，检查是否已存在
	if req.Email != "" && req.Email != user.Email {
		var existingUser models.User
		if err := s.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
			return nil, utils.NewAppError(409, "Email already exists")
		}
		user.Email = req.Email
	}

	if err := s.db.Save(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) UpdateAvatar(id uint, avatarPath string) (*models.User, error) {
	user, err := s.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	// 删除旧头像
	if user.Avatar != "" {
		oldAvatarPath := filepath.Join(s.config.Upload.Path, filepath.Base(user.Avatar))
		if _, err := os.Stat(oldAvatarPath); err == nil {
			os.Remove(oldAvatarPath)
		}
	}

	user.Avatar = avatarPath
	if err := s.db.Save(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetUserList(query models.UserQuery) (*models.UserListResponse, error) {
	var users []models.User
	var total int64

	db := s.db.Model(&models.User{})

	// 搜索条件
	if query.Search != "" {
		search := "%" + query.Search + "%"
		db = db.Where("username LIKE ? OR email LIKE ?", search, search)
	}

	if query.Role != "" {
		db = db.Where("role = ?", models.UserRole(query.Role))
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	if query.Page == 0 {
		query.Page = 1
	}
	if query.PageSize == 0 {
		query.PageSize = 10
	}

	offset := (query.Page - 1) * query.PageSize
	db = db.Offset(offset).Limit(query.PageSize)

	// 获取用户列表
	if err := db.Find(&users).Error; err != nil {
		return nil, err
	}

	// 转换为响应格式
	userResponses := make([]models.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = models.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			Avatar:    user.Avatar,
			CreatedAt: user.CreatedAt,
		}
	}

	totalPages := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))

	return &models.UserListResponse{
		Users:      userResponses,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *UserService) UpdateUserRole(id uint, role models.UserRole, currentUserID uint) (*models.User, error) {
	// 不能修改自己的角色
	if id == currentUserID {
		return nil, utils.NewAppError(400, "Cannot change your own role")
	}

	user, err := s.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	user.Role = role
	if err := s.db.Save(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) SaveAvatar(file *multipart.FileHeader, userID uint) (string, error) {
	// 检查文件大小
	if file.Size > s.config.Upload.MaxSize {
		return "", utils.NewAppError(400, fmt.Sprintf("File too large. Maximum size is %dMB", s.config.Upload.MaxSize>>20))
	}

	// 检查文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := strings.Split(s.config.Upload.AllowedExt, ",")
	validExt := false
	for _, allowedExt := range allowedExts {
		if "."+allowedExt == ext {
			validExt = true
			break
		}
	}
	if !validExt {
		return "", utils.NewAppError(400, "Invalid file type")
	}

	// 创建保存目录
	if err := os.MkdirAll(s.config.Upload.Path, 0755); err != nil {
		return "", err
	}

	// 生成文件名
	filename := fmt.Sprintf("avatar_%d%s", userID, ext)
	filePath := filepath.Join(s.config.Upload.Path, filename)

	// 保存文件
	if err := utils.SaveUploadedFile(file, filePath); err != nil {
		return "", err
	}

	// 返回访问路径
	return s.config.Upload.URLPrefix + "/" + filename, nil
}
