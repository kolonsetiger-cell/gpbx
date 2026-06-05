package callcenter_agent

import (
	"encoding/xml"
	"net/http"
	"strings"

	"gitee.com/kolonse_zhjsh/gpbx/app"
	"gitee.com/kolonse_zhjsh/gpbx/store"
	"github.com/gin-gonic/gin"
)

// 需要修改 freeswitch 配置文件 xml_curl.conf.xml ，修改成以下内容，就可以调用接口
// <configuration name="xml_curl.conf" description="cURL XML Gateway">
//   <bindings>
//     <binding name="ext register">
//       <!-- one or more |-delim of configuration|directory|dialplan -->
//       <param name="gateway-url" value="http://192.168.0.108:8081/api/fs/directory" bindings="directory"/>
//     </binding>
//   </bindings>
// </configuration>

type RegisterRequest struct {
	Hostname           string `form:"hostname"`
	Section            string `form:"section"`
	TagName            string `form:"tag_name"`
	KeyName            string `form:"key_name"`
	KeyValue           string `form:"key_value"`
	EventName          string `form:"Event-Name"`
	CoreUUID           string `form:"Core-UUID"`
	FreeSWITCHHostname string `form:"FreeSWITCH-Hostname"`
	FreeSWITCHIPv4     string `form:"FreeSWITCH-IPv4"`
	Action             string `form:"action"`
	SipProfile         string `form:"sip_profile"`
	SipAuthUsername    string `form:"sip_auth_username"`
	SipAuthRealm       string `form:"sip_auth_realm"`
	SipAuthResponse    string `form:"sip_auth_response"`
	User               string `form:"user"`
	Domain             string `form:"domain"`
	Ip                 string `form:"ip"`
}

// register response
// <document type="freeswitch/xml">
// <section name="directory">
//
//	<domain name="{domain}">
//	    <params>
//	        <param name="dial-string" value="{presence_id=${dialed_user}@${dialed_domain}}${sofia_contact(${dialed_user}@${dialed_domain})}"/>
//	        <param name="jsonrpc-allowed-methods" value="verto"/>
//	    </params>
//	    <groups>
//	        <group name="default">
//	        <users>
//	            <user id="{user}">
//	                <params>
//	                    <param name="password" value="{password}"/>
//	                    <param name="vm-password" value="{password}"/>
//	                </params>
//	                <variables>
//	                    <variable name="toll_allow" value="domestic,international,local"/>
//	                    <variable name="accountcode" value="{user}"/>
//	                    <variable name="user_context" value="default"/>
//	                    <variable name="effective_caller_id_name" value="Extension {user}"/>
//	                    <variable name="effective_caller_id_number" value="{user}"/>
//	                    <variable name="outbound_caller_id_name" value="000000000000"/>
//	                    <variable name="outbound_caller_id_number" value="000000000000"/>
//	                    <variable name="callgroup" value="techsupport"/>
//	                </variables>
//	            </user>
//	        </users>
//	        </group>
//	    </groups>
//	</domain>
//
// </section>
// </document>

type FSDocument struct {
	XMLName xml.Name `xml:"document"`
	Type    string   `xml:"type,attr"`
	Section Section  `xml:"section"`
}

// <section>
type Section struct {
	Name   string `xml:"name,attr"`
	Domain Domain `xml:"domain"`
}

// <domain>
type Domain struct {
	Name   string `xml:"name,attr"`
	Params Params `xml:"params"`
	Groups Groups `xml:"groups"`
}

// <params> 下的多个 <param>
type Params struct {
	Param []Param `xml:"param"`
}

type Param struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// <groups>
type Groups struct {
	Group Group `xml:"group"`
}

// <group>
type Group struct {
	Name  string `xml:"name,attr"`
	Users Users  `xml:"users"`
}

// <users>
type Users struct {
	User User `xml:"user"`
}

// <user>
type User struct {
	ID        string    `xml:"id,attr"`
	Params    Params    `xml:"params"`
	Variables Variables `xml:"variables"`
}

// <variables> 下的多个 <variable>
type Variables struct {
	Variable []Variable `xml:"variable"`
}

type Variable struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

func extension_register_event(c *gin.Context) {
	// body, _ := c.GetRawData()
	// defaultLogger.Debug(ThisModule, "recv %v", string(body))
	var event RegisterRequest
	err := c.ShouldBind(&event)
	if err != nil {
		defaultLogger.Warn(ThisModule, "Check Error : %v", err.Error())
		c.XML(http.StatusBadRequest, nil)
		return
	}
	defaultLogger.Info(ThisModule, "Recv : %v", event)
	if event.Action != "sip_auth" && event.Action != "user_call" {
		defaultLogger.Warn(ThisModule, "Not Support : %v", event.Action)
		c.XML(http.StatusOK, FSDocument{
			Type:    "freeswitch/xml",
			Section: Section{},
		})
		return
	}
	// 查询用户是否存在
	pos := strings.Index(event.User, "-")
	if pos == -1 {
		defaultLogger.Warn(ThisModule, "User Format Error : %v", event.User)
		c.XML(http.StatusOK, FSDocument{
			Type:    "freeswitch/xml",
			Section: Section{},
		})
		return
	}
	tenantId := event.User[0:pos]
	agentId := event.User[pos+1:]

	extension, err := app.GetDefaultApp().GetStoreEngine().QueryExtension(store.Extension{TenantId: tenantId, ExtensionId: agentId})
	if err != nil {
		c.XML(http.StatusOK, FSDocument{
			Type:    "freeswitch/xml",
			Section: Section{},
		})
		return
	}
	if extension.Password == "" {
		extension.Password = app.GetDefaultApp().GetCfg().Child("sip.default_password").GetString()
		// 这里不能计算 HashPassword ，不然给客户端返回的密钥会不一致
		// _ = extension.HashPassword()
	}
	// defaultLogger.Debug(ThisModule, "Password %v ", extension.Password)
	// password := hex.EncodeToString(md5.New().Sum([]byte(extension.Password))[8:24])
	// password = "55714465687a724d31437849722f5561"
	password := extension.Password
	domain := app.GetDefaultApp().GetCfg().Child("sip.domain").GetString()
	defaultLogger.Info(ThisModule, "User %v Domain %v Password %v Check OK", event.User, domain, password)
	resp_xml := FSDocument{
		Type: "freeswitch/xml",
		Section: Section{
			Name: "directory",
			Domain: Domain{
				Name: domain,
				Params: Params{
					Param: []Param{
						{Name: "dial-string", Value: "{presence_id=${dialed_user}@${dialed_domain}}${sofia_contact(${dialed_user}@${dialed_domain})}"},
						{Name: "jsonrpc-allowed-methods", Value: "verto"},
					},
				},
				Groups: Groups{
					Group: Group{
						Name: "default",
						Users: Users{
							User: User{
								ID: event.User,
								Params: Params{
									Param: []Param{
										{Name: "password", Value: password},
										{Name: "vm-password", Value: password},
									},
								},
								Variables: Variables{
									Variable: []Variable{
										{Name: "toll_allow", Value: "domestic,international,local"},
										{Name: "accountcode", Value: event.User},
										{Name: "user_context", Value: "default"},
										{Name: "effective_caller_id_name", Value: "Extension " + event.User},
										{Name: "effective_caller_id_number", Value: event.User},
										{Name: "outbound_caller_id_name", Value: "000000000000"},
										{Name: "outbound_caller_id_number", Value: "000000000000"},
										{Name: "callgroup", Value: "techsupport"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	c.XML(http.StatusOK, resp_xml)
}
func extension_register(router *gin.RouterGroup) {
	router.GET("/fs/directory", extension_register_event)
	router.POST("/fs/directory", extension_register_event)
}

func init() {
	registers = append(registers, extension_register)
}
