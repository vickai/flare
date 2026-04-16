package theme

import (
	"html/template"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/soulteary/flare/config/data"
	"github.com/soulteary/flare/config/define"
	"github.com/soulteary/flare/internal/auth"
	"github.com/soulteary/flare/internal/pool"
	"github.com/soulteary/flare/internal/pages/home" // 自定义 vickai.yml 并生成 HTML
)

func RegisterRouting(e *echo.Echo) {
	e.GET(define.SettingPages.Theme.Path, pageTheme, auth.AuthRequired)
	e.POST(define.SettingPages.Theme.Path, updateThemes, auth.AuthRequired)
}

func updateThemes(c *echo.Context) error {
	var body struct {
		Theme string `form:"theme"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusForbidden, "提交数据缺失")
	}
	data.UpdateThemeName(body.Theme)
	define.UpdatePagePalettes()
	define.ThemeCurrent = body.Theme
	define.ThemePrimaryColor = define.GetThemePrimaryColor(body.Theme)
	return pageTheme(c)
}

func pageTheme(c *echo.Context) error {
	themes := define.ThemePalettes
	options, err := data.GetAllSettingsOptions()
	if err != nil {
		return c.String(http.StatusInternalServerError, "config error")
	}
	locale := options.Locale
	if locale == "" {
		locale = "zh"
	}
	m := pool.GetTemplateMap()
	defer pool.PutTemplateMap(m)

	// --- 🛠️ 关键一步：调用拼装函数并存入 Map ---
	m["VickaiBookmarks"] = home.GenerateVickaiBookmark()
    // ----------------------------------------

	m["Locale"] = locale
	m["DebugMode"] = define.AppFlags.DebugMode
	m["PageInlineStyle"] = define.GetPageInlineStyle()
	m["PageAppearance"] = define.GetAppBodyStyle()
	m["SettingsURI"] = define.RegularPages.Settings.Path
	m["PageName"] = "Theme"
	m["SettingPages"] = define.SettingPages
	m["Themes"] = themes
	m["OptionTitle"] = options.Title
	m["OptionFooter"] = template.HTML(options.Footer)
	return c.Render(http.StatusOK, "settings.html", m)
}
