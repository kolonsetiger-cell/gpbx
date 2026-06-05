package callcenter_manager

import (
	"net/http"
	"strconv"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/gin-gonic/gin"
)

// records: T[]
// current: number
// size: number
// total: number
type PageResponse struct {
	Records []any `json:"records"`
	Current int64 `json:"current"`
	Size    int64 `json:"size"`
	Total   int64 `json:"total"`
}

type RequestTenant struct {
	TenantId      string `json:"tenantId"`
	TenantName    string `json:"tenantName"`
	DefaultNumber string `json:"default_number"`
}

func api_config_tenant_create(c *gin.Context) {
	var tenant RequestTenant
	if err := c.ShouldBindJSON(&tenant); err != nil {
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "Invalid request"})
		return
	}
	if err := app.GetDefaultApp().GetStoreEngine().StoreTenant(store.Tenant{
		TenantId:   tenant.TenantId,
		TenantName: tenant.TenantName,
		CreateTime: time.Now().UnixMilli(),
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v tenant: %v", c.Request.URL.Path, tenant)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func api_config_tenant_query(c *gin.Context) {
	current, _ := strconv.Atoi(c.Query("current"))
	size, _ := strconv.Atoi(c.Query("size"))
	defaultLogger.Info(ThisModule, "%v current: %d, size: %d", c.Request.URL.Path, current, size)
	count := app.GetDefaultApp().GetStoreEngine().CountTenant()
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
	records, _ := app.GetDefaultApp().GetStoreEngine().QueryTenantOfPage(offset, int64(size))
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

func api_config_tenant_update(c *gin.Context) {
	var tenant RequestTenant
	if err := c.ShouldBindJSON(&tenant); err != nil {
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "Invalid request"})
		return
	}
	if err := app.GetDefaultApp().GetStoreEngine().StoreTenant(store.Tenant{
		TenantId:      tenant.TenantId,
		TenantName:    tenant.TenantName,
		DefaultNumber: tenant.DefaultNumber,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v tenant: %v", c.Request.URL.Path, tenant)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func api_config_tenant_delete(c *gin.Context) {
	tenantId := c.Param("tenantId")
	if err := app.GetDefaultApp().GetStoreEngine().DeleteTenant(store.Tenant{
		TenantId: tenantId,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v tenantId: %s", c.Request.URL.Path, tenantId)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

// 注册租户路由
func tenant_registers(router *gin.RouterGroup) {
	// 注册租户路由
	tenant := router.Group("/tenant")
	tenant.GET("/list", api_config_tenant_query)
	tenant.POST("/create", api_config_tenant_create)
	tenant.DELETE("/delete/:tenantId", api_config_tenant_delete)
	tenant.PUT("/update", api_config_tenant_update)
}

func init() {
	registers = append(registers, tenant_registers)
}
