package home

import (
	"os" // 新增 os 包用于读取本地文件
	"gopkg.in/yaml.v2" // 新增 解析 yml 文件
	"net" // 探测在线状态
	"strconv" // 探测在线状态
	"time" // 时间包
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


// 内部函数：高性能 TCP 探测
func isVickaiAlive(ip string, port int) bool {
	// 1. 基础检查
	if ip == "" {
		return false
	}

	// 2. 默认值处理：如果 YAML 没填 port，则默认为 80
	actualPort := port
	if actualPort <= 0 {
		actualPort = 80
	}

	// 3. 拼接地址
	address := net.JoinHostPort(ip, strconv.Itoa(actualPort)) // 👈 推荐使用标准库 JoinHostPort，更安全

	// 4. 执行探测
	conn, err := net.DialTimeout("tcp", address, 200*time.Millisecond)
	if err == nil {
		defer conn.Close() // 使用 defer 确保即使后续逻辑复杂也会关闭连接
		return true
	}
	return false
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
	//   ip:   探测地址
	//   prot：探测端口
	var appsData struct {
		// 解析 YAML (使用 VickaiBookmark 结构 /config/model/bookmark.go)
		Items []model.VickaiBookmark `yaml:"items"`
	}
	if err := yaml.Unmarshal(buf, &appsData); err != nil {
		return template.HTML("vickai-bookmarks.yml 格式解析错误")
	}


	// 2.1. 并发探测状态
	var wg sync.WaitGroup
	statusMap := make(map[string]bool)
	var mu sync.Mutex

	for _, app := range appsData.Items {
		if app.IP != "" {
			wg.Add(1)
			// 建议将端口逻辑拉平：YAML没写是0，函数内部转80
			go func(targetIP string, targetPort int) {
				defer wg.Done()

				// 执行探测
				alive := isVickaiAlive(targetIP, targetPort)

				// 构造唯一的 Key (确保写入和读取时完全一致)
				key := targetIP + ":" + strconv.Itoa(targetPort)

				mu.Lock()
				statusMap[key] = alive
				mu.Unlock()
			}(app.IP, app.Port)
		}
	}
	wg.Wait()


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

		// 确定状态灯类名
		statusClass := ""
		statusDisabled :=""
		statusTips :=""
		if app.IP != "" {
			// 读取时必须使用和上面写入时完全一样的 Key 构造逻辑
			checkKey := app.IP + ":" + strconv.Itoa(app.Port)

			mu.Lock() // 虽然 Wait 结束了，但 map 读取建议保持规范
			isAlive := statusMap[checkKey]
			mu.Unlock()

			if isAlive {
				statusClass = "status-online"
			} else {
				statusClass = "status-offline"
				statusDisabled ="disabled"
				statusTips ="<i>服务离线</i>"
			}
		}

		b.WriteString(`<div class="vickai-container" data-id="`)
		b.WriteString(app.Icon)
		b.WriteString(`"><a ` + target + `rel="noopener" href="`)
		b.WriteString(templateURL)
		b.WriteString(`" class="vickai-item ` + statusDisabled + ` clearfix" title="`)
		b.WriteString(app.Name)
		b.WriteString(`"><div class="vickai-icon">`)
		b.WriteString(templateIcon)
		b.WriteString(`</div><div class="vickai-text"><p class="vickai-title">`)
		b.WriteString(app.Name)
		b.WriteString(`</p><p class="vickai-desc">`)
		b.WriteString(desc)
		b.WriteString(`</p></div>`)
		// 注入状态灯
		if statusClass != "" {
			b.WriteString(`<span class="vickai-dot ` + statusClass + `"></span>`)
			b.WriteString(statusTips)
		}
		b.WriteString(`</a></div>`)
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
	//   ip:   探测地址
	//   prot：探测端口
	var appsData struct {
		// 解析 YAML (使用 VickaiBookmark 结构 /config/model/bookmark.go)
		Items []model.VickaiBookmark `yaml:"items"`
	}

	if err := yaml.Unmarshal(buf, &appsData); err != nil {
		return template.HTML("vickai-nav.yml 格式解析错误")
	}


	// 2.1. 并发探测状态
	var wg sync.WaitGroup
	statusMap := make(map[string]bool)
	var mu sync.Mutex

	for _, app := range appsData.Items {
		if app.IP != "" {
			wg.Add(1)
			// 建议将端口逻辑拉平：YAML没写是0，函数内部转80
			go func(targetIP string, targetPort int) {
				defer wg.Done()

				// 执行探测
				alive := isVickaiAlive(targetIP, targetPort)

				// 构造唯一的 Key (确保写入和读取时完全一致)
				key := targetIP + ":" + strconv.Itoa(targetPort)

				mu.Lock()
				statusMap[key] = alive
				mu.Unlock()
			}(app.IP, app.Port)
		}
	}
	wg.Wait()


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
		} else if options.IconMode == "FILLINguokG" {
			templateIcon = fn.GetYandexFavicon(app.URL, mdi.GetIconByName(app.Icon))
		}

		// 拼装 HTML 结构
		target := ""
		if options.OpenAppNewTab { target = `target="_blank" ` }

		// 确定状态灯类名
		statusClass := ""
		statusDisabled :=""
		if app.IP != "" {
			// 读取时必须使用和上面写入时完全一样的 Key 构造逻辑
			checkKey := app.IP + ":" + strconv.Itoa(app.Port)

			mu.Lock() // 虽然 Wait 结束了，但 map 读取建议保持规范
			isAlive := statusMap[checkKey]
			mu.Unlock()

			if isAlive {
				statusClass = "status-online"
			} else {
				statusClass = "status-offline"
				statusDisabled =` class="disabled" `
				desc = "服务离线: " + desc
			}
		}


		b.WriteString(`<h3 ` + statusDisabled + ` data-id="`)
		b.WriteString(app.Icon)
		b.WriteString(`"><a ` + target + `rel="noopener" href="`)
		b.WriteString(templateURL)
		b.WriteString(`" title="`)
		b.WriteString(desc)
		b.WriteString(`">`)
		b.WriteString(templateIcon)
		b.WriteString(`<span>`)
		b.WriteString(app.Name)
		b.WriteString(`</span></a>`)
		// 注入状态灯
		if statusClass != "" {
			b.WriteString(`<span class="vickai-dot ` + statusClass + `"></span>`)
		}
		b.WriteString(`</h3>`)
	}
	b.WriteString(`</div>`)
	return template.HTML(b.String())
}

// GenerateVickaiService 处理外部 vickai-services.yml 并生成 HTML
func GenerateVickaiService() template.HTML {
	// 1. 自动寻址（兼容 Docker 和本地）
	paths := []string{"/app/vickai-services.yml", "./vickai-services.yml"}
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
	// - name: 示例服务
	//   ip:   探测地址
	//   prot：探测端口
	var appsData struct {
		// 解析 YAML (使用 VickaiBookmark 结构 /config/model/bookmark.go)
		Items []model.VickaiService `yaml:"items"`
	}

	if err := yaml.Unmarshal(buf, &appsData); err != nil {
		return template.HTML("vickai-services.yml 格式解析错误")
	}


	// 2.1. 并发探测状态
	var wg sync.WaitGroup
	statusMap := make(map[string]bool)
	var mu sync.Mutex

	for _, app := range appsData.Items {

		if app.IP != "" {
			wg.Add(1)
			// 建议将端口逻辑拉平：YAML没写是0，函数内部转80
			go func(targetIP string, targetPort int) {
				defer wg.Done()

				// 执行探测
				alive := isVickaiAlive(targetIP, targetPort)

				// 构造唯一的 Key (确保写入和读取时完全一致)
				key := targetIP + ":" + strconv.Itoa(targetPort)

				mu.Lock()
				statusMap[key] = alive
				mu.Unlock()
			}(app.IP, app.Port)
		}
	}
	wg.Wait()


	// 3. 获取系统配置并准备 Builder
	//options, _ := data.GetAllSettingsOptions()
	b := builderPool.Get().(*strings.Builder)
	b.Reset()
	defer builderPool.Put(b)

	// 4. 【核心拼装】
	b.WriteString(`<ul>`)
	b.WriteString(`<li class="clearfix">`)
	b.WriteString(`<span class="vickai-dot-title">状态</span>`)
	b.WriteString(`<span class="vickai-name-title">名称</span>`)
	b.WriteString(`<span class="vickai-ip-title">地址</span>`)
	b.WriteString(`<span class="vickai-port-title">端口</span>`)
	b.WriteString(`</li>`)
	for _, app := range appsData.Items {
		// 确定状态灯类名
		statusClass := ""
		statusClassLi :=  ` class="clearfix"`
		if app.IP != "" {
			// 读取时必须使用和上面写入时完全一样的 Key 构造逻辑
			checkKey := app.IP + ":" + strconv.Itoa(app.Port)

			mu.Lock() // 虽然 Wait 结束了，但 map 读取建议保持规范
			isAlive := statusMap[checkKey]
			mu.Unlock()

			if isAlive {
				statusClass = "status-online"
				statusClassLi = ` class="online clearfix"`
			} else {
				statusClass = "status-offline"
				statusClassLi = ` class="offline clearfix"`
			}
		}

		b.WriteString(`<li` + statusClassLi + `>`)
		// 注入状态灯
		if statusClass != "" {
			b.WriteString(`<span class="vickai-dot ` + statusClass + `"><i></i></span>`)
		} else {
			b.WriteString(`<span class="vickai-dot"></span>`)
		}
		b.WriteString(`<span class="vickai-name"> ` + app.Name + `</span>`)
		b.WriteString(`<span class="vickai-ip"> ` + app.IP + `</span>`)
		b.WriteString(`<span class="vickai-port"> ` + strconv.Itoa(app.Port) + `</span>`)
		b.WriteString(`</li>`)
	}
	b.WriteString(`</ul>`)
	return template.HTML(b.String())
}