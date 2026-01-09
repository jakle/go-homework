package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"gin-examples/project/models"
	"gin-examples/project/services"
	"gin-examples/project/utils"
)

type UserHandler struct {
	userService *services.UserService
	jwtSecret   []byte
}

func NewUserHandler(userService *services.UserService, jwtSecret []byte) *UserHandler {
	return &UserHandler{
		userService: userService,
		jwtSecret:   jwtSecret,
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, parseValidationErrors(err))
		return
	}

	user, err := h.userService.CreateUser(req)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		Avatar:    user.Avatar,
		CreatedAt: user.CreatedAt,
	})
}

func (h *UserHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, parseValidationErrors(err))
		return
	}

	user, err := h.userService.Authenticate(req.Username, req.Password)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	token, err := utils.GenerateToken(h.jwtSecret, user.ID, user.Username, string(user.Role))
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, gin.H{
		"token": token,
		"user": models.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			Avatar:    user.Avatar,
			CreatedAt: user.CreatedAt,
		},
	})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.userService.GetUserByID(userID.(uint))
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		Avatar:    user.Avatar,
		CreatedAt: user.CreatedAt,
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, parseValidationErrors(err))
		return
	}

	user, err := h.userService.UpdateUser(userID.(uint), req)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		Avatar:    user.Avatar,
		CreatedAt: user.CreatedAt,
	})
}

//作业开始

// UploadAvatar 上传用户头像
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Avatar file is required")
		return
	}

	//上传图片
	avatarPath, err := h.userService.SaveAvatar(file, userID.(uint))
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	//更新用户头像
	user, err := h.userService.UpdateAvatar(userID.(uint), avatarPath)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		Avatar:    user.Avatar,
		CreatedAt: user.CreatedAt,
	})
}

// GetUserList 获取用户列表（需要管理员权限）
func (h *UserHandler) GetUserList(c *gin.Context) {
	userRole, exists := c.Get("userRole")
	if !exists || userRole != string(models.RoleAdmin) {
		utils.Error(c, http.StatusForbidden, "Access denied. Admin role required")
		return
	}

	var query models.UserQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.ValidationError(c, parseValidationErrors(err))
		return
	}

	userList, err := h.userService.GetUserList(query)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, userList)
}

// UpdateUserRole 更新用户角色（需要管理员权限）
func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	userRole, exists := c.Get("userRole")
	if !exists || userRole != string(models.RoleAdmin) {
		utils.Error(c, http.StatusForbidden, "Access denied. Admin role required")
		return
	}

	currentUserID, _ := c.Get("userID")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req struct {
		Role models.UserRole `json:"role" binding:"required,oneof=user admin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, parseValidationErrors(err))
		return
	}

	user, err := h.userService.UpdateUserRole(uint(id), req.Role, currentUserID.(uint))
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		Avatar:    user.Avatar,
		CreatedAt: user.CreatedAt,
	})
}

//作业结束

func parseValidationErrors(err error) map[string]string {
	errors := make(map[string]string)

	// 检查是否为验证错误
	if bindErr, ok := err.(gin.Error); ok {
		// 处理 gin 的验证错误
		if bindErr.Err != nil {
			errors["general"] = bindErr.Err.Error()
		} else {
			errors["general"] = bindErr.Error()
		}
		return errors
	}

	// 如果是其他类型的错误，直接返回通用错误
	errors["general"] = err.Error()
	return errors
}
