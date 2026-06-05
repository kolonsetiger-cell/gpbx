package callcenter_manager

import (
	"net/http"
	"strconv"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/gin-gonic/gin"
)

type RequestTenantNumber struct {
	Number   string `json:"number"`
	TenantId string `json:"tenantId"`
	Action   int    `json:"action"`
	WayType  string `json:"wayType"`
	Way      string `json:"way"`
	RobotID  string `json:"robotId"`
}

func api_config_number_create(c *gin.Context) {
	var number RequestTenantNumber
	if err := c.ShouldBindJSON(&number); err != nil {
		c.JSON(400, Response{
			Code: 400,
			Msg:  "参数错误:" + err.Error(),
		})
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		return
	}
	if err := app.GetDefaultApp().GetStoreEngine().StoreTenantNumber(store.TenantNumber{
		Number:     number.Number,
		TenantId:   number.TenantId,
		Action:     int64(number.Action),
		WayType:    number.WayType,
		Way:        number.Way,
		RobotID:    number.RobotID,
		CreateTime: time.Now().Unix(),
	}); err != nil {
		c.JSON(500, Response{
			Code: 500,
			Msg:  "创建失败",
		})
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		return
	}
	defaultLogger.Info(ThisModule, "%v number: %v", c.Request.URL.Path, number)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "创建成功",
	})
}

func api_config_number_query(c *gin.Context) {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil || current < 1 {
		current = 1
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil || size < 1 {
		size = 10
	}
	defaultLogger.Info(ThisModule, "%v current: %d, size: %d", c.Request.URL.Path, current, size)
	count := app.GetDefaultApp().GetStoreEngine().CountTenantNumber()
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
	records, _ := app.GetDefaultApp().GetStoreEngine().QueryTenantNumberOfPage(offset, int64(size))
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

func api_config_number_update(c *gin.Context) {
	var number RequestTenantNumber
	if err := c.ShouldBindJSON(&number); err != nil {
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "Invalid request"})
		return
	}
	if err := app.GetDefaultApp().GetStoreEngine().StoreTenantNumber(store.TenantNumber{
		Number:   number.Number,
		TenantId: number.TenantId,
		Action:   int64(number.Action),
		WayType:  number.WayType,
		Way:      number.Way,
		RobotID:  number.RobotID,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v number: %v", c.Request.URL.Path, number)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func api_config_number_delete(c *gin.Context) {
	numberID := c.Param("id")
	if numberID == "" {
		c.JSON(400, Response{
			Code: 400,
			Msg:  "参数错误: numberID 不能为空",
		})
		return
	}
	if err := app.GetDefaultApp().GetStoreEngine().DeleteTenantNumber(store.TenantNumber{Number: numberID}); err != nil {
		c.JSON(500, Response{
			Code: 500,
			Msg:  "删除失败",
		})
		return
	}
	c.JSON(200, Response{
		Code: 200,
		Msg:  "删除成功",
	})
}

// 注册租户路由
func number_registers(router *gin.RouterGroup) {
	number := router.Group("/number")
	{
		number.GET("/list", api_config_number_query)
		number.POST("/create", api_config_number_create)
		number.PUT("/update", api_config_number_update)
		number.DELETE("/delete/:id", api_config_number_delete)
	}
}

func init() {
	registers = append(registers, number_registers)
}
