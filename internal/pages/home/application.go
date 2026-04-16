package home

import (
	"os" // 新增 os 包用于读取本地文件
	"gopkg.in/yaml.v2" // 新增 解析 yml 文件
	"html/template"
	"strings"
	"sync"

	"github.com/soulteary/flare/config/data"
	"github.com/soulteary/flare/config/model"
	"github.com/soulteary/flare/internal/fn"
	"github.com/soulteary/flare/internal/resources/mdi"
)

var builderPool = sync.Pool{
	New: func() any { return &strings.Builder{} },
}

func GenerateApplicationsTemplate(filter string, options *model.Application) template.HTML {
	if options == nil {
		op, err := data.GetAllSettingsOptions()
		if err != nil {
			op = model.Application{}
		}
		options = &op
	}
	appsData, err := data.LoadFavoriteBookmarks()
	if err != nil {
		return template.HTML("")
	}
	b, ok := builderPool.Get().(*strings.Builder)
	if !ok {
		b = &strings.Builder{}
	}
	b.Reset()
	defer builderPool.Put(b)

	n := len(appsData.Items)
	parseApps := make([]model.Bookmark, 0, n)
	for _, app := range appsData.Items {
		app.URL = fn.ParseDynamicUrl(app.URL)
		parseApps = append(parseApps, app)
	}

	var apps []model.Bookmark
	if filter != "" {
		apps = make([]model.Bookmark, 0, n)
	}

	if filter != "" {
		filterLower := strings.ToLower(filter)
		for _, bookmark := range parseApps {
			if strings.Contains(strings.ToLower(bookmark.Name), filterLower) || strings.Contains(strings.ToLower(bookmark.URL), filterLower) || strings.Contains(strings.ToLower(bookmark.Desc), filterLower) {
				apps = append(apps, bookmark)
			}
		}
	} else {
		apps = parseApps
	}

	for _, app := range apps {
		desc := app.Desc
		if desc == "" {
			desc = app.URL
		}
		templateURL := app.URL
		if strings.HasPrefix(app.URL, "chrome-extension://") || options.EnableEncryptedLink {
			templateURL = "/redir/url?go=" + data.Base64EncodeUrl(app.URL)
		}
		templateIcon := mdi.GetIconByName(app.Icon)
		if strings.HasPrefix(app.Icon, "http://") || strings.HasPrefix(app.Icon, "https://") {
			templateIcon = `<img src="` + app.Icon + `"/>`
		} else if app.Icon != "" {
			templateIcon = mdi.GetIconByName(app.Icon)
		} else if options.IconMode == "FILLING" {
			templateIcon = fn.GetYandexFavicon(app.URL, mdi.GetIconByName(app.Icon))
		}
		if options.OpenAppNewTab {
			b.WriteString(`<div class="app-container" data-id="`)
			b.WriteString(app.Icon)
			b.WriteString(`"><a target="_blank" rel="noopener" href="`)
			b.WriteString(templateURL)
			b.WriteString(`" class="app-item" title="`)
			b.WriteString(app.Name)
			b.WriteString(`"><div class="app-icon">`)
			b.WriteString(templateIcon)
			b.WriteString(`</div><div class="app-text"><p class="app-title">`)
			b.WriteString(app.Name)
			b.WriteString(`</p><p class="app-desc">`)
			b.WriteString(desc)
			b.WriteString(`</p></div></a></div>`)
		} else {
			b.WriteString(`<div class="app-container" data-id="`)
			b.WriteString(app.Icon)
			b.WriteString(`"><a rel="noopener" href="`)
			b.WriteString(templateURL)
			b.WriteString(`" class="app-item" title="`)
			b.WriteString(app.Name)
			b.WriteString(`"><div class="app-icon">`)
			b.WriteString(templateIcon)
			b.WriteString(`</div><div class="app-text"><p class="app-title">`)
			b.WriteString(app.Name)
			b.WriteString(`</p><p class="app-desc">`)
			b.WriteString(desc)
			b.WriteString(`</p></div></a></div>`)
		}
	}
	return template.HTML(b.String())
}


// GenerateVickaiBookmark 处理外部 vickai-bookmarks.yml 并生成 HTML
func GenerateVickaiBookmark() template.HTML {
	// 1. 自动寻址（兼容 Docker 和本地）
	paths := []string{"/app/vickai-bookmarks.yml", "./vickai-bookmarks.yml"}
	var buf []byte
	var err error
	for _, p := range paths {
		buf, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		return template.HTML("") // 读不到文件则不显示
	}

	// 2. 解析 YAML (结构必须带 items 层级)
	// items:
	// - name: 示例链接
	//   link: https://link.example.com
	//   icon: ChatGPT.svg
	//   desc: 链接描述文本
	var appsData struct {
		Items []model.Bookmark `yaml:"items"`
	}
	if err := yaml.Unmarshal(buf, &appsData); err != nil {
		return template.HTML("vickai-bookmarks.yml 格式解析错误")
	}

	// 3. 获取系统配置并准备 Builder
	options, _ := data.GetAllSettingsOptions()
	b := builderPool.Get().(*strings.Builder)
	b.Reset()
	defer builderPool.Put(b)

	// 4. 【核心拼装】完全同步作者的图标与 URL 处理逻辑
	for _, app := range appsData.Items {
		app.URL = fn.ParseDynamicUrl(app.URL)
		desc := app.Desc
		if desc == "" { desc = app.URL }

		// 处理重定向和加密链接
		templateURL := app.URL
		if strings.HasPrefix(app.URL, "chrome-extension://") || options.EnableEncryptedLink {
			templateURL = "/redir/url?go=" + data.Base64EncodeUrl(app.URL)
		}

		// 处理图标（MDI / HTTP图片 / Favicon）
		templateIcon := ""
		if strings.HasPrefix(app.Icon, "http://") || strings.HasPrefix(app.Icon, "https://") {
			templateIcon = `<img src="` + app.Icon + `"/>`
		} else if app.Icon != "" {
			templateIcon = mdi.GetIconByName(app.Icon)
		} else if options.IconMode == "FILLING" {
			templateIcon = fn.GetYandexFavicon(app.URL, mdi.GetIconByName(app.Icon))
		}

		// 拼装 HTML 结构
		target := ""
		if options.OpenAppNewTab { target = `target="_blank" ` }

		b.WriteString(`<div class="vickai-container" data-id="`)
		b.WriteString(app.Icon)
		b.WriteString(`"><a ` + target + `rel="noopener" href="`)
		b.WriteString(templateURL)
		b.WriteString(`" class="vickai-item clearfix" title="`)
		b.WriteString(app.Name)
		b.WriteString(`"><div class="vickai-icon">`)
		b.WriteString(templateIcon)
		b.WriteString(`</div><div class="vickai-text"><p class="vickai-title">`)
		b.WriteString(app.Name)
		b.WriteString(`</p><p class="vickai-desc">`)
		b.WriteString(desc)
		b.WriteString(`</p></div></a></div>`)
	}
	return template.HTML(b.String())
}


// GenerateVickaiNav 处理外部 vickai-nav.yml 并生成 HTML
func GenerateVickaiNav() template.HTML {
	// 1. 自动寻址（兼容 Docker 和本地）
	paths := []string{"/app/vickai-nav.yml", "./vickai-nav.yml"}
	var buf []byte
	var err error
	for _, p := range paths {
		buf, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		return template.HTML("") // 读不到文件则不显示
	}

	// 2. 解析 YAML (结构必须带 items 层级)
	// items:
	// - name: 示例链接
	//   link: https://link.example.com
	//   icon: ChatGPT.svg
	//   desc: 链接描述文本
	var appsData struct {
		Items []model.Bookmark `yaml:"items"`
	}
	if err := yaml.Unmarshal(buf, &appsData); err != nil {
		return template.HTML("vickai-nav.yml 格式解析错误")
	}

	// 3. 获取系统配置并准备 Builder
	options, _ := data.GetAllSettingsOptions()
	b := builderPool.Get().(*strings.Builder)
	b.Reset()
	defer builderPool.Put(b)

	// 4. 【核心拼装】完全同步作者的图标与 URL 处理逻辑
	b.WriteString(`<div id="vickai-nav">`)
	for _, app := range appsData.Items {
		app.URL = fn.ParseDynamicUrl(app.URL)
		desc := app.Desc
		if desc == "" { desc = app.URL }

		// 处理重定向和加密链接
		templateURL := app.URL
		if strings.HasPrefix(app.URL, "chrome-extension://") || options.EnableEncryptedLink {
			templateURL = "/redir/url?go=" + data.Base64EncodeUrl(app.URL)
		}

		// 处理图标（MDI / HTTP图片 / Favicon）
		templateIcon := ""
		if strings.HasPrefix(app.Icon, "http://") || strings.HasPrefix(app.Icon, "https://") {
			templateIcon = `<img src="` + app.Icon + `"/>`
		} else if app.Icon != "" {
			templateIcon = mdi.GetIconByName(app.Icon)
		} else if options.IconMode == "FILLING" {
			templateIcon = fn.GetYandexFavicon(app.URL, mdi.GetIconByName(app.Icon))
		}

		// 拼装 HTML 结构
		target := ""
		if options.OpenAppNewTab { target = `target="_blank" ` }

		b.WriteString(`<h3 data-id="`)
		b.WriteString(app.Icon)
		b.WriteString(`"><a ` + target + `rel="noopener" href="`)
		b.WriteString(templateURL)
		b.WriteString(`" title="`)
		b.WriteString(desc)
		b.WriteString(`">`)
		b.WriteString(templateIcon)
		b.WriteString(`<span>`)
		b.WriteString(app.Name)
		b.WriteString(`</span></a></h3>`)
	}
	b.WriteString(`</div>`)
	return template.HTML(b.String())
}