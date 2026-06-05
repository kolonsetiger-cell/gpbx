package callcenter_manager

import (
	"net/http"
	"strconv"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/gin-gonic/gin"
)

type RequestTeleGramUser struct {
	Username    string `json:"username"`
	TenantId    string `json:"tenantId"`
	AuthTime    int64  `json:"authTime"`
	ExpireTime  int64  `json:"expireTime"`
	BindScript  string `json:"bindScript"`
	BindNumbers string `json:"bindNumbers"`
	IvrId       string `json:"ivrId"`
}

func api_config_telegram_user_create(c *gin.Context) {
	var req RequestTeleGramUser
	if err := c.ShouldBindJSON(&req); err != nil {
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "Invalid request"})
		return
	}
	if err := app.GetDefaultApp().GetStoreEngine().StoreTeleGramUser(store.TeleGramUser{
		Username:    req.Username,
		TenantId:    req.TenantId,
		AuthTime:    req.AuthTime,
		ExpireTime:  req.ExpireTime,
		BindScript:  req.BindScript,
		BindNumbers: req.BindNumbers,
		IvrId:       req.IvrId,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v telegram_user: %v", c.Request.URL.Path, req)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func api_config_telegram_user_query(c *gin.Context) {
	current, _ := strconv.Atoi(c.Query("current"))
	size, _ := strconv.Atoi(c.Query("size"))
	defaultLogger.Info(ThisModule, "%v current: %d, size: %d", c.Request.URL.Path, current, size)
	count := app.GetDefaultApp().GetStoreEngine().CountTeleGramUser()
	if count == 0 {
		c.JSON(200, Response{
			Code: 200,
			Msg:  "Success",
			Data: PageResponse{
				Records: []any{},
				Current: 1,
				Size:    int64(size),
				Total:   count,
			},
		})
		return
	}
	offset := max(int64(current)*int64(size)-int64(size), 0)
	records, _ := app.GetDefaultApp().GetStoreEngine().QueryTeleGramUserOfPage(offset, int64(size))
	defaultLogger.Info(ThisModule, "%v records: %v", c.Request.URL.Path, records)
	response := PageResponse{
		Records: []any{},
		Current: int64(current),
		Size:    int64(size),
		Total:   count,
	}
	for _, record := range records {
		response.Records = append(response.Records, record)
	}
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
		Data: response,
	})
}

func api_config_telegram_user_update(c *gin.Context) {
	var req RequestTeleGramUser
	if err := c.ShouldBindJSON(&req); err != nil {
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "Invalid request"})
		return
	}
	if err := app.GetDefaultApp().GetStoreEngine().StoreTeleGramUser(store.TeleGramUser{
		Username:    req.Username,
		TenantId:    req.TenantId,
		AuthTime:    req.AuthTime,
		ExpireTime:  req.ExpireTime,
		BindScript:  req.BindScript,
		BindNumbers: req.BindNumbers,
		IvrId:       req.IvrId,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v telegram_user: %v", c.Request.URL.Path, req)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func api_config_telegram_user_delete(c *gin.Context) {
	username := c.Param("username")
	if err := app.GetDefaultApp().GetStoreEngine().DeleteTeleGramUser(store.TeleGramUser{
		Username: username,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v username: %s", c.Request.URL.Path, username)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

// 注册 Telegram 用户路由
func telegram_user_registers(router *gin.RouterGroup) {
	tg := router.Group("/telegram_user")
	tg.GET("/list", api_config_telegram_user_query)
	tg.POST("/create", api_config_telegram_user_create)
	tg.DELETE("/delete/:username", api_config_telegram_user_delete)
	tg.PUT("/update", api_config_telegram_user_update)
}

func init() {
	registers = append(registers, telegram_user_registers)
}
