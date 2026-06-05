package callcenter_manager

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	ID       uint     `json:"userId"`
	Username string   `json:"userName"`
	Roles    []string `json:"roles"`
	Email    string   `json:"email"`
}

// Response 登录响应
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, username string) (string, error) {
	// 设置过期时间
	expire_hours := app.GetDefaultApp().GetCfg().Child("jwt.expire_hours").GetInt()
	secret := app.GetDefaultApp().GetCfg().Child("jwt.secret").GetString()
	expireTime := time.Now().Add(time.Duration(expire_hours) * time.Hour)

	// 创建声明
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	// 创建令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名令牌
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GenerateRefreshToken 生成刷新令牌
func GenerateRefreshToken(userID uint, username string) (string, error) {
	// 设置过期时间（通常比访问令牌更长）
	expire_hours := app.GetDefaultApp().GetCfg().Child("jwt.refresh_expire_hours").GetInt()
	secret := app.GetDefaultApp().GetCfg().Child("jwt.secret").GetString()
	expireTime := time.Now().Add(time.Duration(expire_hours) * time.Hour)

	// 创建声明
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	// 创建令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用刷新密钥签名令牌
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "请求参数错误"})
		return
	}
	user, err := app.GetDefaultApp().GetStoreEngine().QueryUser(store.User{Username: req.Username})
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "用户不存在"})
		return
	}
	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "用户名或密码错误"})
		return
	}

	// // 生成令牌
	token, err := GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "服务器内部错误"})
		return
	}
	refreshToken, err := GenerateRefreshToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "服务器内部错误"})
		return
	}

	// 构建响应
	response := &Response{
		Code: 200,
		Msg:  "登录成功",
		Data: LoginResponse{
			Token:        token,
			RefreshToken: refreshToken,
		},
	}
	c.JSON(http.StatusOK, response)
}

// 刷新令牌
func refresh(c *gin.Context) {
	authorization := c.GetHeader("Authorization")
	if authorization == "" {
		c.JSON(http.StatusUnauthorized, Response{Code: 401, Msg: "未授权"})
		return
	}
	tokenString := strings.Replace(authorization, "Bearer ", "", 1)
	claims, err := ParseToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, Response{Code: 401, Msg: "无效的令牌"})
		return
	}
	_, err = app.GetDefaultApp().GetStoreEngine().QueryUser(store.User{Username: claims.Username})
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "用户不存在"})
		return
	}
	// 生成新的令牌
	token, err := GenerateToken(claims.UserID, claims.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "服务器内部错误"})
		return
	}
	refreshToken, err := GenerateRefreshToken(claims.UserID, claims.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "服务器内部错误"})
		return
	}
	// 构建响应
	response := &Response{
		Code: 200,
		Msg:  "刷新令牌成功",
		Data: LoginResponse{
			Token:        token,
			RefreshToken: refreshToken,
		},
	}
	c.JSON(http.StatusOK, response)
}

func ParseToken(tokenString string) (*Claims, error) {
	secret := app.GetDefaultApp().GetCfg().Child("jwt.secret").GetString()
	// 解析令牌
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	// 验证令牌
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// 获取用户信息
func getUserInfo(c *gin.Context) {
	authorization := c.GetHeader("Authorization")
	if authorization == "" {
		c.JSON(http.StatusUnauthorized, Response{Code: 401, Msg: "未授权"})
		return
	}
	tokenString := strings.Replace(authorization, "Bearer ", "", 1)
	claims, err := ParseToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, Response{Code: 401, Msg: "无效的令牌"})
		return
	}
	user, err := app.GetDefaultApp().GetStoreEngine().QueryUser(store.User{Username: claims.Username})
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "用户不存在"})
		return
	}
	response := &Response{
		Code: 200,
		Msg:  "获取用户信息成功",
		Data: UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Roles:    []string{user.Roles},
			Email:    "",
		},
	}
	c.JSON(http.StatusOK, response)
}

func auth(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", login)
		auth.POST("/refresh", refresh)
	}
	router.GET("/user/info", getUserInfo)
}

func init() {
	registers = append(registers, auth)
}
