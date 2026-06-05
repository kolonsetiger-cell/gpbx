package callcenter_agent

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
	TenantID string `json:"tenantId" binding:"required"`
	AgentId  string `json:"agentId" binding:"required"`
}

type SipInfo struct {
	SipWsServer string `json:"sipWsServer"`
	SipDomain   string `json:"sipDomain"`
	SipUser     string `json:"sipUser"`
	SipPass     string `json:"sipPass"`
}

type UserResponse struct {
	AgentServer   string   `json:"agentServer"`
	SipInfo       SipInfo  `json:"sipInfo"`
	DisplayNumber string   `json:"displayNumber"`
	Numbers       []string `json:"numbers"`
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
	TenantId string `json:"tenant_id"`
	AgentId  string `json:"agent_id"`
	jwt.RegisteredClaims
}

func GenerateToken(TenantId string, AgentId string) (string, error) {
	// 设置过期时间
	expire_hours := app.GetDefaultApp().GetCfg().Child("jwt.expire_hours").GetInt()
	secret := app.GetDefaultApp().GetCfg().Child("jwt.secret").GetString()
	expireTime := time.Now().Add(time.Duration(expire_hours) * time.Hour)

	// 创建声明
	claims := &Claims{
		TenantId: TenantId,
		AgentId:  AgentId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   TenantId + "-" + AgentId,
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
func GenerateRefreshToken(TenantId string, AgentId string) (string, error) {
	// 设置过期时间（通常比访问令牌更长）
	expire_hours := app.GetDefaultApp().GetCfg().Child("jwt.refresh_expire_hours").GetInt()
	secret := app.GetDefaultApp().GetCfg().Child("jwt.secret").GetString()
	expireTime := time.Now().Add(time.Duration(expire_hours) * time.Hour)

	// 创建声明
	claims := &Claims{
		TenantId: TenantId,
		AgentId:  AgentId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   TenantId + "-" + AgentId,
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
	user, err := app.GetDefaultApp().GetStoreEngine().QueryAgent(store.Agent{TenantId: req.TenantID, AgentId: req.AgentId})
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "用户不存在"})
		return
	}

	// // 生成令牌
	token, err := GenerateToken(user.TenantId, user.AgentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "服务器内部错误"})
		return
	}
	refreshToken, err := GenerateRefreshToken(user.TenantId, user.AgentId)
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
	_, err = app.GetDefaultApp().GetStoreEngine().QueryAgent(store.Agent{TenantId: claims.TenantId, AgentId: claims.AgentId})
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "用户不存在"})
		return
	}
	// 生成新的令牌
	token, err := GenerateToken(claims.TenantId, claims.AgentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "服务器内部错误"})
		return
	}
	refreshToken, err := GenerateRefreshToken(claims.TenantId, claims.AgentId)
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
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
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
	tenant, err := app.GetDefaultApp().GetStoreEngine().QueryTenant(store.Tenant{TenantId: claims.TenantId})
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "租户不存在"})
		return
	}
	user, err := app.GetDefaultApp().GetStoreEngine().QueryAgent(store.Agent{TenantId: claims.TenantId, AgentId: claims.AgentId})
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "用户不存在"})
		return
	}
	extension, err := app.GetDefaultApp().GetStoreEngine().QueryExtension(store.Extension{TenantId: claims.TenantId, ExtensionId: user.ExtensionId})
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "分机不存在"})
		return
	}
	if extension.Password == "" {
		extension.Password = app.GetDefaultApp().GetCfg().Child("sip.default_password").GetString()
		// _ = extension.HashPassword()
	}

	db_numbers, err := app.GetDefaultApp().GetStoreEngine().QueryTenantNumbers(store.TenantNumber{TenantId: claims.TenantId})
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "租户号码不存在"})
		return
	}
	var numbers []string
	for _, db_number := range db_numbers {
		numbers = append(numbers, db_number.Number)
	}

	var displayNumber string = user.DisplayNumber
	if displayNumber == "" {
		displayNumber = tenant.DefaultNumber
	}

	if displayNumber == "" {
		if len(numbers) > 0 {
			displayNumber = numbers[0]
		}
	}

	agentServer := app.GetDefaultApp().GetCfg().Child("agent.ws_addr").GetString()
	response := &Response{
		Code: 200,
		Msg:  "获取用户信息成功",
		Data: UserResponse{
			AgentServer: agentServer,
			SipInfo: SipInfo{
				SipWsServer: app.GetDefaultApp().GetCfg().Child("sip.ws_addr").GetString(),
				SipDomain:   app.GetDefaultApp().GetCfg().Child("sip.domain").GetString(),
				SipUser:     user.TenantId + "-" + user.ExtensionId,
				SipPass:     extension.Password,
			},
			DisplayNumber: displayNumber,
			Numbers:       numbers,
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
	router.GET("/agent/info", getUserInfo)
}

func init() {
	registers = append(registers, auth)
}
