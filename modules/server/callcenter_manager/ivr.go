package callcenter_manager

import (
	"net/http"
	"strconv"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/gin-gonic/gin"
)

type RequestIvr struct {
	IvrID string `json:"ivrId"`
	Type  string `json:"type"`
	Path  string `json:"path"`
	Args  string `json:"args"`
}

func api_config_ivr_create(c *gin.Context) {
	var ivr RequestIvr
	if err := c.ShouldBindJSON(&ivr); err != nil {
		c.JSON(400, Response{
			Code: 400,
			Msg:  err.Error(),
		})
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		return
	}
	if err := app.GetDefaultApp().GetStoreEngine().StoreIvr(store.Ivr{
		IvrID:      ivr.IvrID,
		Type:       ivr.Type,
		Path:       ivr.Path,
		Args:       ivr.Args,
		CreateTime: time.Now().UnixMilli(),
	}); err != nil {
		c.JSON(400, Response{
			Code: 400,
			Msg:  err.Error(),
		})
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		return
	}
	defaultLogger.Info(ThisModule, "%v ivr: %v", c.Request.URL.Path, ivr)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func api_config_ivr_query(c *gin.Context) {
	current, _ := strconv.Atoi(c.Query("current"))
	size, _ := strconv.Atoi(c.Query("size"))
	defaultLogger.Info(ThisModule, "%v current: %d, size: %d", c.Request.URL.Path, current, size)
	count := app.GetDefaultApp().GetStoreEngine().CountIvr()
	if count == 0 {
		c.JSON(200, Response{
			Code: 200,
			Msg:  "Success",
		})
		return
	}
	offset := max(int64(current)*int64(size)-int64(size), 0)
	records, _ := app.GetDefaultApp().GetStoreEngine().QueryIvrOfPage(offset, int64(size))
	defaultLogger.Info(ThisModule, "%v records: %v", c.Request.URL.Path, records)
	response := PageResponse{
		Records: []any{},
		Current: int64(current),
		Size:    int64(size),
		Total:   count,
	}
	for _, record := range records {
		response.Records = append(response.Records, map[string]any{
			"ivrId":      record.IvrID,
			"type":       record.Type,
			"path":       record.Path,
			"args":       record.Args,
			"createTime": record.CreateTime,
		})
	}
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
		Data: response,
	})
}

func api_config_ivr_update(c *gin.Context) {
	var ivr RequestIvr
	if err := c.ShouldBindJSON(&ivr); err != nil {
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "Invalid request"})
		return
	}
	if err := app.GetDefaultApp().GetStoreEngine().StoreIvr(store.Ivr{
		IvrID: ivr.IvrID,
		Type:  ivr.Type,
		Path:  ivr.Path,
		Args:  ivr.Args,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v ivr: %v", c.Request.URL.Path, ivr)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func api_config_ivr_delete(c *gin.Context) {
	ivrID := c.Param("id")
	if err := app.GetDefaultApp().GetStoreEngine().DeleteIvr(store.Ivr{
		IvrID: ivrID,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v ivrID: %s", c.Request.URL.Path, ivrID)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func ivr_registers(router *gin.RouterGroup) {
	ivr := router.Group("/ivr")
	{
		ivr.GET("/list", api_config_ivr_query)
		ivr.POST("/create", api_config_ivr_create)
		ivr.PUT("/update/:id", api_config_ivr_update)
		ivr.DELETE("/delete/:id", api_config_ivr_delete)
	}
}

func init() {
	registers = append(registers, ivr_registers)
}
