package auth

import (
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	session "github.com/labstack/echo-contrib/v5/session"
	"github.com/labstack/echo/v5"

	"github.com/soulteary/flare/config/define"
)

const (
	SESSION_KEY_USER_NAME  = "USER_NAME"
	SESSION_KEY_LOGIN_DATE = "LOGIN_TIME"
)

// sessionName is set by RequestHandle and used by session.Get. Prefer passing name via RequestHandleSessionName.
var sessionName string

// RequestHandleSessionName returns the session name for the given cookie name and port (for testing or explicit wiring).
func RequestHandleSessionName(cookieName string, port int) string {
	return fmt.Sprintf("%s_%d", cookieName, port)
}

// ... 保持 import 不变 ...

var store *sessions.CookieStore // 1. 将 store 定义为指针，不要在包级别初始化

func RequestHandle(e *echo.Echo) {
	// 2. 确保在 RequestHandle 执行时才计算名字和初始化 Store
	sessionName = RequestHandleSessionName(define.AppFlags.CookieName, define.AppFlags.Port)

	if !define.AppFlags.DisableLoginMode {
		// 3. 此时 define.AppFlags 已经被 cmd.Parse() 填充完毕
		secret := []byte(define.AppFlags.CookieSecret)

		// 容错处理：如果密钥依然为空，给一个硬编码的保底值防止报错
		if len(secret) == 0 {
			secret = []byte("vickai-default-32-byte-secret-!!)@")
		}

		store = sessions.NewCookieStore(secret)

		// 4. 设置配置
		store.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		}

		e.Use(session.Middleware(store))
		e.POST(define.MiscPages.Login.Path, login)
		e.POST(define.MiscPages.Logout.Path, logout)
	}
}

var commonText = `<a href="` + define.SettingPages.Others.Path + `">返回重试</a></p><p>或前往 <a href="https://github.com/soulteary/docker-flare/issues/" target="_blank">https://github.com/soulteary/docker-flare/issues/</a> 反馈使用中的问题，谢谢！`
var internalErrorInput = []byte(`<html><p>请填写正确的用户名和密码 ` + commonText + `</html>`)
var internalErrorEmpty = []byte(`<html><p>用户名或密码不能为空 ` + commonText + `</html>`)
var internalErrorSave = []byte(`<html><p>程序内部错误，保存登陆状态失败 ` + commonText + `</html>`)

func AuthRequired(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if !define.AppFlags.DisableLoginMode {
			sess, err := session.Get(sessionName, c)
			if err != nil {
				return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
			}
			user := sess.Values[SESSION_KEY_USER_NAME]
			if user == nil {
				return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
			}
		}
		return next(c)
	}
}

func CheckUserIsLogin(c *echo.Context) bool {
	if !define.AppFlags.DisableLoginMode {
		sess, err := session.Get(sessionName, c)
		if err != nil {
			return false
		}
		user := sess.Values[SESSION_KEY_USER_NAME]
		return user != nil
	}
	return true
}

func GetUserName(c *echo.Context) string {
	if !define.AppFlags.DisableLoginMode {
		sess, err := session.Get(sessionName, c)
		if err != nil {
			return ""
		}
		if v, ok := sess.Values[SESSION_KEY_USER_NAME].(string); ok {
			return v
		}
	}
	return ""
}

func GetUserLoginDate(c *echo.Context) string {
	if !define.AppFlags.DisableLoginMode {
		sess, err := session.Get(sessionName, c)
		if err != nil {
			return ""
		}
		if v, ok := sess.Values[SESSION_KEY_LOGIN_DATE].(string); ok {
			return v
		}
	}
	return ""
}

func login(c *echo.Context) error {
	// 1. 获取 Session（不因旧 Cookie 报错而中断）
	sess, err := session.Get(sessionName, c)
	if err != nil {
		log.Printf("[auth] 获取 Session 异常(通常为旧密钥过期): %v", err)
		sess, _ = session.Get(sessionName, c) // 强制获取新会话
	}

	username := c.FormValue("username")
	password := c.FormValue("password")

	// 2. 统一使用 TrimSpace (比 Trim(s, " ") 更全面，能处理换行等)
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return c.HTMLBlob(http.StatusBadRequest, internalErrorEmpty)
	}

	// 3. 账号密码比对
	if subtle.ConstantTimeCompare([]byte(username), []byte(define.AppFlags.User)) != 1 ||
		subtle.ConstantTimeCompare([]byte(password), []byte(define.AppFlags.Pass)) != 1 {
		log.Printf("[auth] 登录失败: 用户名或密码错误 (User: %s)", username)
		return c.HTMLBlob(http.StatusBadRequest, internalErrorInput)
	}

	// 4. 写入 Session
	sess.Values[SESSION_KEY_USER_NAME] = username
	sess.Values[SESSION_KEY_LOGIN_DATE] = time.Now().Format("2006年01月02日 15:04:05 CST")

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Printf("[auth] Session 保存失败: %v", err)
		return c.HTMLBlob(http.StatusBadRequest, internalErrorSave)
	}

	log.Printf("[auth] 用户登录成功: %s", username)
	return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
}

func logout(c *echo.Context) error {
	sess, err := session.Get(sessionName, c)
	if err != nil {
		return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
	}
	if sess.Values[SESSION_KEY_USER_NAME] == nil {
		return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
	}
	delete(sess.Values, SESSION_KEY_USER_NAME)
	delete(sess.Values, SESSION_KEY_LOGIN_DATE)

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return c.HTMLBlob(http.StatusBadRequest, internalErrorSave)
	}
	return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
}
