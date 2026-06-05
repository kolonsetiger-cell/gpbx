package callcenter_manager

import (
	"net/http"
	"strconv"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/gin-gonic/gin"
)

type RequestAgent struct {
	TenantId      string `json:"tenantId"`
	AgentId       string `json:"agentId"`
	AgentName     string `json:"agentName"`
	ExtensionId   string `json:"extensionId"`
	DisplayNumber string `json:"displayNumber"`
}

func api_config_agent_create(c *gin.Context) {
	var agent RequestAgent
	if err := c.ShouldBindJSON(&agent); err != nil {
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "Invalid request"})
		return
	}
	if err := app.GetDefaultApp().GetStoreEngine().StoreAgent(store.Agent{
		TenantId:      agent.TenantId,
		AgentId:       agent.AgentId,
		AgentName:     agent.AgentName,
		ExtensionId:   agent.ExtensionId,
		DisplayNumber: agent.DisplayNumber,
		CreateTime:    time.Now().UnixMilli(),
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v agent: %v", c.Request.URL.Path, agent)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func api_config_agent_query(c *gin.Context) {
	current, _ := strconv.Atoi(c.Query("current"))
	size, _ := strconv.Atoi(c.Query("size"))
	defaultLogger.Info(ThisModule, "%v current: %d, size: %d", c.Request.URL.Path, current, size)
	count := app.GetDefaultApp().GetStoreEngine().CountAgent()
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
	records, _ := app.GetDefaultApp().GetStoreEngine().QueryAgentOfPage(offset, int64(size))
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

func api_config_agent_update(c *gin.Context) {
	var agent RequestAgent
	if err := c.ShouldBindJSON(&agent); err != nil {
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "Invalid request"})
		return
	}
	if err := app.GetDefaultApp().GetStoreEngine().StoreAgent(store.Agent{
		TenantId:      agent.TenantId,
		AgentId:       agent.AgentId,
		AgentName:     agent.AgentName,
		ExtensionId:   agent.ExtensionId,
		DisplayNumber: agent.DisplayNumber,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v agent: %v", c.Request.URL.Path, agent)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func api_config_agent_delete(c *gin.Context) {
	agentId := c.Param("agentId")
	if err := app.GetDefaultApp().GetStoreEngine().DeleteAgent(store.Agent{
		AgentId: agentId,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v agentId: %s", c.Request.URL.Path, agentId)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

// 注册坐席路由
func agent_registers(router *gin.RouterGroup) {
	agent := router.Group("/agent")
	agent.GET("/list", api_config_agent_query)
	agent.POST("/create", api_config_agent_create)
	agent.DELETE("/delete/:agentId", api_config_agent_delete)
	agent.PUT("/update", api_config_agent_update)
}

func init() {
	registers = append(registers, agent_registers)
}
