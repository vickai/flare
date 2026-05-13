package home

import (
	"context"      // 👈 新增：用于处理命令超时
	"encoding/json" // 👈 新增：用于解析 Tailscale 的 JSON 输出
	"os" // 新增 os 包用于读取本地文件
	"os/exec"      // 👈 新增：用于执行 tailscale 和 ping 命令
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


// 内部函数：智能在线探测 (TCP + Ping 回退)
func isVickaiAlive(ip string, port int) bool {
	if ip == "" {
		return false
	}

	// 1. 如果填了端口，优先进行 TCP 探测 (最准确)
	if port > 0 {
		address := net.JoinHostPort(ip, strconv.Itoa(port))
		// 缩短超时时间到 300ms，平衡探测速度和网络波动
		conn, err := net.DialTimeout("tcp", address, 300*time.Millisecond)
		if err == nil {
			_ = conn.Close() // 显式关闭，习惯更好
			return true
		}
		// 如果 TCP 失败了，不要急着判死刑，尝试 Ping (可能是服务挂了但机器还活着)
	}

	// 2. 回退机制：执行系统 Ping 探测
	// 依赖 Dockerfile 中安装的 iputils-ping
	// -c 1: 发送 1 个包
	// -W 1: 等待 1 秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", ip)
	err := cmd.Run()

	// 如果命令执行成功（Exit Code 0），说明设备在网
	return err == nil
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

// GenerateVickaiService 生成带有分类导航、TCP探测及动态Tailscale状态的HTML内容
func GenerateVickaiService() template.HTML {
	// 1. 自动寻址与读取 YAML
	paths := []string{"/app/vickai-services.yml", "./vickai-services.yml"}
	var buf []byte
	var err error
	for _, p := range paths {
		buf, err = os.ReadFile(p)
		if err == nil { break }
	}
	if err != nil { return template.HTML("") }

	// 2. 解析支持分类的 YAML 结构
	var data struct {
		Groups []model.VickaiServiceGroup `yaml:"groups"`
	}
	if err := yaml.Unmarshal(buf, &data); err != nil {
		return template.HTML("vickai-services.yml 格式解析错误")
	}

	// 3. 状态获取准备
	var wg sync.WaitGroup
	statusMap := make(map[string]bool)
	var mu sync.Mutex
	var tsData *model.TailscaleStatus

	// 3.1 遍历分类进行探测
	for _, group := range data.Groups {
		if group.Category == "Tailscale" {
			// 如果是 Tailscale 分类，通过 Socket 获取实时 JSON 状态
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			cmd := exec.CommandContext(ctx, "tailscale", "--socket=/var/run/tailscale/tailscaled.sock", "status", "--json")
			if output, err := cmd.Output(); err == nil {
				_ = json.Unmarshal(output, &tsData)
			}
			cancel()
		} else {
			// 普通分类，并发执行 TCP/Ping 高性能探测
			for _, app := range group.Items {
				if app.IP != "" {
					wg.Add(1)
					go func(targetIP string, targetPort int) {
						defer wg.Done()
						alive := isVickaiAlive(targetIP, targetPort)
						key := targetIP + ":" + strconv.Itoa(targetPort)
						mu.Lock()
						statusMap[key] = alive
						mu.Unlock()
					}(app.IP, app.Port)
				}
			}
		}
	}
	wg.Wait()

	// 4. 准备 Builder (从对象池获取以优化内存)
	b := builderPool.Get().(*strings.Builder)
	b.Reset()
	defer builderPool.Put(b)

	// 5. 内部辅助：清洗 Tailscale 名称 (优先取 DNSName 前缀)
	cleanTSName := func(dnsName, hostName string) string {
		if dnsName != "" {
			trimmed := strings.TrimSuffix(dnsName, ".")
			parts := strings.Split(trimmed, ".")
			if len(parts) > 0 && parts[0] != "" {
				return parts[0]
			}
		}
		return hostName
	}

	// --- 6. 第一阶段：生成顶部导航栏 ---
	b.WriteString(`<div class="vickai-service-nav">`)
	for i, group := range data.Groups {
		categoryID := "v-cat-" + strconv.Itoa(i)
		b.WriteString(`<a href="#` + categoryID + `">` + group.Category + `</a>`)
	}
	b.WriteString(`</div>`)

	// --- 7. 第二阶段：生成详细列表内容 ---
	for i, group := range data.Groups {
		categoryID := "v-cat-" + strconv.Itoa(i)

		if group.Category == "Tailscale" && tsData != nil {
			// 7.1 处理动态 Tailscale 节点
			// 写入表头
			b.WriteString(`<div class="vickai-ts-service-group" id="` + categoryID + `">`)
			b.WriteString(`<h3 class="vickai-category-title"><span>` + group.Category + `</span></h3>`)
			b.WriteString(`<ul>`)
			b.WriteString(`<li class="vickai-ts-table-header clearfix">`)
			b.WriteString(`<span class="vickai-ts-dot-title">状态</span>`)
			b.WriteString(`<span class="vickai-ts-name-title">主机名</span>`)
			b.WriteString(`<span class="vickai-ts-os-title">操作系统</span>`)
			b.WriteString(`<span class="vickai-ts-ip-title">Tailscale IP</span>`)
			b.WriteString(`<span class="vickai-ts-mode-title">连接信息</span>`)
			b.WriteString(`<span class="vickai-ts-time-title">最后在线</span>`)
			b.WriteString(`</li>`)
			// 项目循环
			// 先渲染宿主机 (Self)
			sName := cleanTSName(tsData.Self.DNSName, tsData.Self.HostName)
			renderTSNode(b, sName, tsData.Self.OS, tsData.Self.Online, tsData.Self.TailscaleIPs, tsData.Self.Relay, tsData.Self.CurAddr, tsData.Self.LastSeen)

			// 再渲染 Peer 成员
			for _, p := range tsData.Peer {
				pName := cleanTSName(p.DNSName, p.HostName)
				renderTSNode(b, pName, p.OS, p.Online, p.TailscaleIPs, p.Relay, p.CurAddr, p.LastSeen)
			}
		} else {
			// 7.2 处理普通配置项目
			// 写入表头
			b.WriteString(`<div class="vickai-service-group" id="` + categoryID + `">`)
			b.WriteString(`<h3 class="vickai-category-title"><span>` + group.Category + `</span></h3>`)
			b.WriteString(`<ul>`)
			b.WriteString(`<li class="vickai-table-header clearfix">`)
			b.WriteString(`<span class="vickai-dot-title">状态</span>`)
			b.WriteString(`<span class="vickai-name-title">服务名称</span>`)
			b.WriteString(`<span class="vickai-ip-title">服务地址</span>`)
			b.WriteString(`<span class="vickai-port-title">检测端口</span>`)
			b.WriteString(`</li>`)
			// 项目循环
			for _, app := range group.Items {
				statusClass := "status-offline"
				statusClassLi := ` class="clearfix"`
				if app.IP != "" {
					checkKey := app.IP + ":" + strconv.Itoa(app.Port)
					mu.Lock()
					isAlive := statusMap[checkKey]
					mu.Unlock()

					if isAlive {
						statusClass = "status-online"; statusClassLi = ` class="online clearfix"`
					} else {
						statusClass = "status-offline"; statusClassLi = ` class="offline clearfix"`
					}
				}

				b.WriteString(`<li` + statusClassLi + `>`)
				b.WriteString(`<span class="vickai-dot ` + statusClass + `"><i></i></span>`)
				b.WriteString(`<span class="vickai-name">` + app.Name + `</span>`)
				b.WriteString(`<span class="vickai-ip">` + app.IP + `</span>`)
				b.WriteString(`<span class="vickai-port">` + strconv.Itoa(app.Port) + `</span>`)
				b.WriteString(`</li>`)
			}
		}
		b.WriteString(`</ul></div>`)
	}

	return template.HTML(b.String())
}

// renderTSNode 辅助函数：渲染单个 Tailscale 节点的 HTML 行
func renderTSNode(b *strings.Builder, name, os string, online bool, ips []string, relay string, curAddr string, lastSeen time.Time) {
	statusClass := "status-offline"
	statusLi := ` class="offline clearfix"`
	// 如果 Online 为真或 CurAddr 有值（隧道已建立），判定为在线
	if online || curAddr != "" {
		statusClass = "status-online"
		statusLi = ` class="online clearfix"`
	}

	b.WriteString(`<li` + statusLi + `>`)
	b.WriteString(`<span class="vickai-ts-dot ` + statusClass + `"><i></i></span>`)
	b.WriteString(`<span class="vickai-ts-name">` + name + `</span>`)
	b.WriteString(`<span class="vickai-ts-os">` + os + `</span>`)

	ip := "N/A"
	if len(ips) > 0 { ip = ips[0] }
	b.WriteString(`<span class="vickai-ts-ip">` + ip + `</span>`)

	// 计算连接模式
	connMode := "离线"
	if curAddr != "" {
		connMode = "直连"
	} else if relay != "" {
		connMode = "中继:" + relay
	} else if online {
		connMode = "待机"
	}

	// 格式化上次在线时间
	timeStr := formatLastSeen(lastSeen)

	b.WriteString(`<span class="vickai-ts-mode">` + connMode + `</span>`)
	b.WriteString(`<span class="vickai-ts-time">` + timeStr + `</span>`)
	b.WriteString(`</li>`)
}

// formatLastSeen 将 time.Time 转换为易读的相对时间
func formatLastSeen(t time.Time) string {
	if t.IsZero() || t.Year() < 2000 { return "现在" }
	duration := time.Since(t)
	if duration.Minutes() < 1 { return "现在" }
	if duration.Hours() < 1 { return strconv.Itoa(int(duration.Minutes())) + "分钟前" }
	if duration.Hours() < 24 { return strconv.Itoa(int(duration.Hours())) + "小时前" }
	return strconv.Itoa(int(duration.Hours()/24)) + "天前"
}