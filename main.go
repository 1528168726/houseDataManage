package main

import (
	"backEnd"
	"errors"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

func main() {
	//重置日志输出
	go redirectLogFile()
	go redirectErrorFile()
	time.Sleep(time.Second)

	//设置release模式
	//gin.SetMode(gin.ReleaseMode)
	//创建路由，设置session
	router := gin.Default()
	gin.New()
	gin.DisableConsoleColor()
	store := cookie.NewStore([]byte(sessions.DefaultKey + "pxj"))
	store.Options(sessions.Options{MaxAge: 0})
	router.Use(sessions.Sessions("user", store))

	router.LoadHTMLGlob("htmlFile/*")
	addRouter(router)

	router.Run(":18080")
}

func addRouter(router *gin.Engine) {
	router.GET("/", func(c *gin.Context) {
		if checkPassword(c) {
			c.Redirect(http.StatusSeeOther, "/home")
		} else {
			c.Redirect(http.StatusSeeOther, "/login")
		}
	})

	router.GET("/login", func(c *gin.Context) {
		if !checkPassword(c) {
			c.HTML(http.StatusOK, "login.html", gin.H{})
		} else {
			c.Redirect(http.StatusSeeOther, "/logined")
		}
	})

	router.POST("/login", func(c *gin.Context) {
		user := c.PostForm("user")
		pwd := c.PostForm("pwd")
		if err := backEnd.CheckPSW(user, pwd); err == nil {
			session := sessions.Default(c)
			session.Set("user", user)
			session.Set("pwd", pwd)
			session.Save()
			c.Redirect(http.StatusSeeOther, "/home")
		} else {
			fmt.Println("user:", user, "密码错误")
			c.HTML(http.StatusOK, "login.html", gin.H{
				"pwdErr": "密码错误",
			})
		}
	})

	router.GET("/logined", func(c *gin.Context) {
		c.HTML(http.StatusOK, "logined.html", gin.H{})
	})

	router.GET("/logOut", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Clear()
		session.Save()
		c.Redirect(http.StatusSeeOther, "/login")
	})

	router.GET("/home", func(c *gin.Context) {
		if checkPassword(c) {
			searchkey := getSearchKey(c)

			data, allCount, err := backEnd.SearchData(searchkey)
			if err != nil {
				fmt.Fprintln(os.Stderr, time.Now().String(), err)
				c.HTML(http.StatusOK, "home.html", gin.H{
					"error": "错误，" + err.Error(),
				})
				return
			}
			estateAddrs, err := backEnd.GetAllEstateAddr()
			if err != nil {
				fmt.Fprintln(os.Stderr, time.Now().String(), err)
				c.HTML(http.StatusOK, "home.html", gin.H{
					"error": "错误，" + err.Error(),
				})
				return
			}
			c.HTML(http.StatusOK, "home.html", gin.H{
				"user":         getUser(c),
				"EstateAddrs":  estateAddrs,
				"curUrl":       url.QueryEscape(c.Request.URL.String()),
				"byEstateAddr": urlSetOrder(c, backEnd.I_estateAddr),
				"byArea":       urlSetOrder(c, backEnd.I_area),
				"byPrice":      urlSetOrder(c, backEnd.I_price),
				"byUnitPrice":  urlSetOrder(c, backEnd.I_unitPrice),
				"houseInfo":    data,
				"curPage":      getCurPage(c),
				"allPage":      getAllPage(allCount, backEnd.SearchLenLimit),
				"firstPage":    urlSet(c, "page", "1"),
				"prePage":      urlSet(c, "page", strconv.Itoa(getCurPage(c)-1)),
				"nextPage":     urlSet(c, "page", strconv.Itoa(getCurPage(c)+1)),
				"lastPage":     urlSet(c, "page", strconv.Itoa(getAllPage(allCount, backEnd.SearchLenLimit))),
				"preShow":      preShow(c),
				"nextShow":     nextShow(c, getAllPage(allCount, backEnd.SearchLenLimit)),
				"rootShow":     rootShow(c),
			})
		} else {
			c.Redirect(http.StatusSeeOther, "/login")
		}
	})

	router.GET("/detail", func(c *gin.Context) {
		if checkPassword(c) {
			houseId, _ := strconv.Atoi(c.DefaultQuery("houseId", "0"))
			data, err := backEnd.GetData(houseId)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				c.HTML(http.StatusOK, "detail.html", gin.H{
					"error": err,
				})
				return
			}
			//设置isSell输出
			isSell := func() string {
				if data.IsSell {
					return "出售"
				} else {
					return "出租"
				}
			}
			//获得上级Url
			preUrl, _ := url.QueryUnescape(c.DefaultQuery("preUrl", "/home"))
			c.HTML(http.StatusOK, "detail.html", gin.H{
				"houseInfo": data,
				"isSell":    isSell(),
				"user":      getUser(c),
				"delete":    `/delete?houseId=` + strconv.Itoa(houseId) + "&preUrl=" + url.QueryEscape(c.DefaultQuery("preUrl", "/home")),
				"rootShow":  rootShow(c),
				"preUrl":    preUrl,
			})

		} else {
			c.Redirect(http.StatusSeeOther, "/login")
		}
	})

	router.GET("/delete", func(c *gin.Context) {
		if checkPassword(c) {
			//获得上级Url
			preUrl, _ := url.QueryUnescape(c.DefaultQuery("preUrl", "/home"))

			if getUser(c) == "root" {
				if c.Query("sure") == "true" {
					houseId, _ := strconv.Atoi(c.DefaultQuery("houseId", "0"))
					backEnd.DeleteHouseResource(houseId)
					c.Redirect(http.StatusSeeOther, preUrl)
				} else {
					c.HTML(http.StatusOK, "delete.html", gin.H{
						"sureDelete": urlSet(c, "sure", "true"),
						"rootShow":   rootShow(c),
						"cancel":     preUrl,
					})
				}
			} else {
				c.HTML(http.StatusOK, "delete.html", gin.H{
					"error":    "没有权限删除",
					"rootShow": rootShow(c),
					"cancel":   preUrl,
				})
			}
		} else {
			c.Redirect(http.StatusSeeOther, "/login")
		}
	})

	router.GET("/admin", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/admin/manage")
	})

	router.GET("/admin/:action", func(c *gin.Context) {
		if checkPassword(c) {
			if getUser(c) == "root" {
				var msg string
				var users *[]string
				action := c.Param("action")
				if action == "manage" {
					users, _ = backEnd.GetAllUser()
				} else if action == "addUser" {
					//添加用户
					newUser := c.Query("newUser")
					pwd := c.Query("pwd")
					if len(pwd) < 6 {
						msg = "密码太短"
					} else {
						if err := backEnd.AddUser(newUser, pwd); err != nil {
							msg = err.Error()
						} else {
							msg = "添加成功"
						}
					}
					users, _ = backEnd.GetAllUser()
				} else if action == "userDelete" {
					//删除用户
					user := c.Query("user")
					if user == "root" {
						msg = "不能删除root账户"
					} else if err := backEnd.DeleteUserbyRoot(user); err != nil {
						msg = err.Error()
					}
					users, _ = backEnd.GetAllUser()
				}
				c.HTML(http.StatusOK, "admin.html", gin.H{
					"user":     getUser(c),
					"users":    users,
					"msg":      msg,
					"rootShow": rootShow(c),
				})

			} else {
				c.Redirect(http.StatusSeeOther, "/home")
			}
		} else {
			c.Redirect(http.StatusSeeOther, "/login")
		}
	})

	router.GET("/submitHouseInfo", func(c *gin.Context) {
		if checkPassword(c) {
			estates, err := backEnd.GetAllEstateAddr()
			if err != nil {
				c.String(http.StatusInternalServerError, `can't get estates name`)
				log.Println(`can't get estates name`)
				return
			}
			c.HTML(http.StatusOK, "submitHouseInfo.html", gin.H{
				"EstateAddrs": *estates,
				"user":        getUser(c),
				"rootShow":    rootShow(c),
			})
		} else {
			c.Redirect(http.StatusSeeOther, "/login")
		}
	})

	router.POST("/submitHouseInfo", func(c *gin.Context) {
		if checkPassword(c) {
			info, err := newHouseInfo(c)
			if err == nil && backEnd.IsRepeat(info) {
				err = errors.New("该地址已经被录入")
			}
			if err == nil {
				err = backEnd.InsertHousingResource(info)
			}
			if err != nil {
				c.HTML(http.StatusForbidden, "submitHouseInfoErr.html", gin.H{
					"error":    err.Error(),
					"user":     getUser(c),
					"rootShow": rootShow(c),
				})
				return
			}
			c.HTML(http.StatusOK, "submitHouseInfoOk.html", gin.H{
				"user":     getUser(c),
				"rootShow": rootShow(c),
			})
		} else {
			c.Redirect(http.StatusSeeOther, "/login")
		}

	})

	router.GET("/resetPassword", func(c *gin.Context) {
		if checkPassword(c) {
			c.HTML(http.StatusOK, "resetPassword.html", gin.H{
				"user":     getUser(c),
				"rootShow": rootShow(c),
			})
		} else {
			c.Redirect(http.StatusSeeOther, "/login")
		}
	})

	router.POST("/resetPassword", func(c *gin.Context) {
		if checkPassword(c) {
			session := sessions.Default(c)
			oldPwd := session.Get("pwd").(string)

			if c.PostForm("newPwd1") != c.PostForm("newPwd2") {
				c.HTML(http.StatusOK, "resetPassword.html", gin.H{
					"user":     getUser(c),
					"rootShow": rootShow(c),

					"newMsg": `两次密码不一致`,
				})
				return
			}
			if len(c.PostForm("newPwd1")) < 6 {
				c.HTML(http.StatusOK, "resetPassword.html", gin.H{
					"user":     getUser(c),
					"rootShow": rootShow(c),
					"newMsg":   `密码过短`,
				})
				return
			}
			if oldPwd != c.PostForm("oldPwd") {
				c.HTML(http.StatusOK, "resetPassword.html", gin.H{
					"user":     getUser(c),
					"rootShow": rootShow(c),
					"newMsg":   `旧密码错误`,
				})
				return
			}
			err := backEnd.ResetPwdByRoot(session.Get("user").(string), c.PostForm("newPwd1"))
			if err != nil {
				c.HTML(http.StatusOK, "resetPassword.html", gin.H{
					"user":     getUser(c),
					"rootShow": rootShow(c),
					"newMsg":   "重置密码失败" + err.Error(),
				})
			} else {
				c.HTML(http.StatusOK, "resetPassword.html", gin.H{
					"user":     getUser(c),
					"rootShow": rootShow(c),
					"newMsg":   "重置密码成功",
				})
			}
		} else {
			c.Redirect(http.StatusSeeOther, "/login")
		}
	})
	return
}

func newHouseInfo(c *gin.Context) (*backEnd.HousingResource, error) {
	info := new(backEnd.HousingResource)

	info.Date = time.Now().String()[0:10]
	session := sessions.Default(c)
	info.Inputer = session.Get("user").(string)

	tmp := c.PostForm("IsSell")
	if tmp == "true" {
		info.IsSell = true
	} else {
		info.IsSell = false
	}

	info.EstateAddr = c.PostForm("EstateAddr")
	if info.EstateAddr == "other" {
		info.EstateAddr = c.PostForm("NewEstateAddr")
		if info.EstateAddr == "" {
			return nil, errors.New("未填写新的楼盘名")
		}
	}

	var err error
	info.BuildingAddr, err = strconv.Atoi(c.PostForm("BuildingAddr"))
	if err != nil {
		return nil, errors.New("BuildingAddr error")
	}

	if c.PostForm("DetailAddr1") == "" {
		info.DetailAddr = c.PostForm("DetailAddr2") + " 层 " + c.PostForm("DetailAddr3")
	} else {
		info.DetailAddr = c.PostForm("DetailAddr1") + "单元 " + c.PostForm("DetailAddr2") + "层 " + c.PostForm("DetailAddr3")
	}

	info.Owner.Name = c.PostForm("OwnerName")
	info.Owner.PhoneNumber = c.PostForm("OwnerPhoneNumber")
	info.Owner.WeChatNumber = c.PostForm("WeChatNumber")

	info.HouseType = c.PostForm("HouseType")
	info.Orientation = c.PostForm("Orientation")

	info.Area, err = strconv.ParseFloat(c.PostForm("Area"), 64)
	if err != nil {
		return nil, errors.New("parse Area error")
	}

	info.Price, err = strconv.ParseFloat(c.PostForm("Price"), 64)
	if err != nil {
		return nil, errors.New("parse Price error")
	}

	info.Remark = c.PostForm("Remark")

	return info, nil

}

func checkPassword(c *gin.Context) bool {
	session := sessions.Default(c)
	id := session.Get("user")
	pwd := session.Get("pwd")

	if id == nil || pwd == nil {
		return false
	}
	log.Println(id)
	err := backEnd.CheckPSW(id.(string), pwd.(string))
	if err == nil {
		return true
	} else if err == backEnd.PwdNotMatch {
		return false
	} else {
		return false
	}
}

func getSearchKey(c *gin.Context) *backEnd.SearchKey {
	key := new(backEnd.SearchKey)

	//get EstateAddr
	if c.Query("searchSelect") == "" {
		key.EstateAddr = ""
	} else if c.Query("searchSelect") == "manual" {
		key.EstateAddr = c.DefaultQuery("manualSearch", "")
	} else {
		key.EstateAddr = c.Query("searchSelect")
	}

	var err error
	//get BuildingAddr
	key.BuildingAddr, err = strconv.Atoi(c.DefaultQuery("BuildingAddr", "0"))
	if err != nil {
		key.BuildingAddr = 0
	}

	//get ListOrder
	i, err := strconv.Atoi(c.DefaultQuery("listOrder", "0"))
	key.ListOrder = backEnd.TypeListOrder(i)
	if err != nil {
		key.ListOrder = backEnd.I_estateAddr
	}

	//get begin
	key.Begin, err = strconv.Atoi(c.DefaultQuery("page", "1"))
	key.Begin = (key.Begin - 1) * backEnd.SearchLenLimit
	if err != nil {
		key.Begin = 0
	}

	return key
}

func getCurPage(c *gin.Context) int {
	curPage, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	return curPage
}

func getAllPage(n int, limit int) int {
	if n%limit != 0 {
		return n/limit + 1
	} else {
		return n / limit
	}
}

func getUser(c *gin.Context) string {
	session := sessions.Default(c)
	return session.Get("user").(string)
}

func urlSetOrder(c *gin.Context, order backEnd.TypeListOrder) string {
	var tOrder int
	//得到正确的排序依据

	tOrder, _ = strconv.Atoi(c.DefaultQuery("listOrder", "0"))
	if int(order) == tOrder {
		if tOrder%2 == 0 {
			tOrder += 1
		} else {
			tOrder -= 1
		}
	} else {
		tOrder = int(order)
	}

	var u url.URL = *c.Request.URL
	value := u.Query()
	value.Set("listOrder", strconv.Itoa(tOrder))
	value.Set("page", "1")
	u.RawQuery = value.Encode()
	return u.String()
}

//url设置键值对
func urlSet(c *gin.Context, key string, value string) string {
	var u url.URL = *c.Request.URL
	values := u.Query()
	values.Set(key, value)
	u.RawQuery = values.Encode()
	return u.String()
}
func rootShow(c *gin.Context) string {
	if getUser(c) == "root" {
		return "display:inline"
	} else {
		return "display:none"
	}
}

func preShow(c *gin.Context) string {
	if getCurPage(c) == 1 {
		return "display:none"
	} else {
		return "display:inline"
	}
}

func nextShow(c *gin.Context, allPage int) string {
	if getCurPage(c) == allPage {
		return "display:none"
	} else {
		return "display:inline"
	}
}
