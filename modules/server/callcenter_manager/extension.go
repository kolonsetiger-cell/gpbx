package callcenter_manager

import (
	"net/http"
	"strconv"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/gin-gonic/gin"
)

type RequestExtension struct {
	TenantId    string `json:"tenantId"`
	ExtensionId string `json:"extensionId"`
	Password    string `json:"password"`
	NetworkIp   string `json:"networkIp"`
	NetworkPort string `json:"networkPort"`
}

func api_config_extension_create(c *gin.Context) {
	var extension RequestExtension
	if err := c.ShouldBindJSON(&extension); err != nil {
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "Invalid request"})
		return
	}
	if extension.Password == "" {
		extension.Password = app.GetDefaultApp().GetCfg().Child("sip.default_password").GetString()
	}
	if err := app.GetDefaultApp().GetStoreEngine().StoreExtension(store.Extension{
		TenantId:    extension.TenantId,
		ExtensionId: extension.ExtensionId,
		Password:    extension.Password,
		CreateTime:  time.Now().UnixMilli(),
		NetworkIP:   extension.NetworkIp,
		NetworkPort: extension.NetworkPort,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v extension: %v", c.Request.URL.Path, extension)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func api_config_extension_query(c *gin.Context) {
	current, _ := strconv.Atoi(c.Query("current"))
	size, _ := strconv.Atoi(c.Query("size"))
	defaultLogger.Info(ThisModule, "%v current: %d, size: %d", c.Request.URL.Path, current, size)
	count := app.GetDefaultApp().GetStoreEngine().CountExtension()
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
	records, _ := app.GetDefaultApp().GetStoreEngine().QueryExtensionOfPage(offset, int64(size))
	defaultLogger.Info(ThisModule, "%v records: %v", c.Request.URL.Path, records)
	response := PageResponse{
		Records: []any{},
		Current: int64(current),
		Size:    int64(size),
		Total:   count,
	}
	for _, record := range records {
		ext := store.ExtensionManagerInstance.GetByNumber(record.TenantId + "-" + record.ExtensionId)
		if ext != nil {
			if ext.IsValid() {
				record.Status = store.EXTENSION_STATUS_ONLINE
			} else {
				record.Status = store.EXTENSION_STATUS_OFFLINE
			}
		} else {
			record.Status = store.EXTENSION_STATUS_OFFLINE
		}
		response.Records = append(response.Records, record)
	}
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
		Data: response,
	})
}

func api_config_extension_delete(c *gin.Context) {
	tenantId := c.Param("tenantId")
	extensionId := c.Param("extensionId")
	if err := app.GetDefaultApp().GetStoreEngine().DeleteExtension(store.Extension{
		TenantId:    tenantId,
		ExtensionId: extensionId,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v extensionId: %s", c.Request.URL.Path, extensionId)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

// 注册分机路由
func extension_registers(router *gin.RouterGroup) {
	extension := router.Group("/extension")
	extension.GET("/list", api_config_extension_query)
	extension.POST("/create", api_config_extension_create)
	extension.DELETE("/delete/:extensionId", api_config_extension_delete)
}

func init() {
	registers = append(registers, extension_registers)
}
