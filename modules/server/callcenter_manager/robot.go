package callcenter_manager

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/gin-gonic/gin"
)

type RequestRobot struct {
	RobotID string `json:"robotId"`
	Target  string `json:"target"`
	Arg     string `json:"arg"`
	Prompt  string `json:"prompt"`
	Welcome string `json:"welcome"`
}

func api_config_robot_create(c *gin.Context) {
	var robot RequestRobot
	if err := c.ShouldBindJSON(&robot); err != nil {
		c.JSON(400, Response{
			Code: 400,
			Msg:  err.Error(),
		})
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		return
	}
	var arg map[string]any
	err := json.Unmarshal([]byte(robot.Arg), &arg)
	if err != nil {
		c.JSON(400, Response{
			Code: 400,
			Msg:  err.Error(),
		})
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		return
	}
	if err := app.GetDefaultApp().GetStoreEngine().StoreRobot(store.Robot{RobotID: robot.RobotID,
		Target: robot.Target, Arg: arg, Welcome: robot.Welcome, Prompt: robot.Prompt,
		CreateTime: time.Now().UnixMilli(),
	}); err != nil {
		c.JSON(400, Response{
			Code: 400,
			Msg:  err.Error(),
		})
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		return
	}
	defaultLogger.Info(ThisModule, "%v robot: %v", c.Request.URL.Path, robot)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func api_config_robot_query(c *gin.Context) {
	current, _ := strconv.Atoi(c.Query("current"))
	size, _ := strconv.Atoi(c.Query("size"))
	defaultLogger.Info(ThisModule, "%v current: %d, size: %d", c.Request.URL.Path, current, size)
	count := app.GetDefaultApp().GetStoreEngine().CountRobot()
	if count == 0 {
		c.JSON(200, Response{
			Code: 200,
			Msg:  "Success",
		})
		return
	}
	offset := max(int64(current)*int64(size)-int64(size), 0)
	records, _ := app.GetDefaultApp().GetStoreEngine().QueryRobotOfPage(offset, int64(size))
	defaultLogger.Info(ThisModule, "%v records: %v", c.Request.URL.Path, records)
	response := PageResponse{
		Records: []any{},
		Current: int64(current),
		Size:    int64(size),
		Total:   count,
	}
	for _, record := range records {
		arg, _ := json.MarshalIndent(record.Arg, "", "  ")
		response.Records = append(response.Records, map[string]any{
			"robotId":    record.RobotID,
			"target":     record.Target,
			"arg":        string(arg),
			"prompt":     record.Prompt,
			"welcome":    record.Welcome,
			"createTime": record.CreateTime,
		})
	}
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
		Data: response,
	})
}

func api_config_robot_update(c *gin.Context) {
	var robot RequestRobot
	if err := c.ShouldBindJSON(&robot); err != nil {
		defaultLogger.Error(ThisModule, "%v Err:%s", c.Request.URL.Path, err)
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "Invalid request"})
		return
	}

	var arg map[string]any
	err := json.Unmarshal([]byte(robot.Arg), &arg)
	if err != nil {
		arg = nil
	}
	if err := app.GetDefaultApp().GetStoreEngine().StoreRobot(store.Robot{
		RobotID: robot.RobotID,
		Target:  robot.Target,
		Prompt:  robot.Prompt,
		Arg:     arg,
		Welcome: robot.Welcome,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v robot: %v", c.Request.URL.Path, robot)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func api_config_robot_delete(c *gin.Context) {
	robotID := c.Param("id")
	if err := app.GetDefaultApp().GetStoreEngine().DeleteRobot(store.Robot{
		RobotID: robotID,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "Internal server error"})
		return
	}
	defaultLogger.Info(ThisModule, "%v robotID: %s", c.Request.URL.Path, robotID)
	c.JSON(200, Response{
		Code: 200,
		Msg:  "Success",
	})
}

func robot_registers(router *gin.RouterGroup) {
	robot := router.Group("/robot")
	{
		robot.GET("/list", api_config_robot_query)
		robot.POST("/create", api_config_robot_create)
		robot.PUT("/update/:id", api_config_robot_update)
		robot.DELETE("/delete/:id", api_config_robot_delete)
	}
}

func init() {
	registers = append(registers, robot_registers)
}
