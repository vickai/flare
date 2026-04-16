package home

import (
	"html/template"

	"github.com/soulteary/flare/config/define"
	"github.com/soulteary/flare/config/model"
	"github.com/soulteary/flare/internal/resources/mdi"
)

func GenerateHelpTemplate() template.HTML {
	apps := []model.Bookmark{}
	apps = append(apps, []model.Bookmark{
		{
			Name: "程序首页",
			URL:  define.RegularPages.Home.Path,
			Icon: "Help-Favicon.svg",
			Desc: "HomeLab 导航",
		},
		{
			Name: "帮助页面",
			URL:  define.RegularPages.Help.Path,
			Icon: "Help-Help.svg",
			Desc: "当前所在页面",
		},
		{
			Name: "程序设置",
			URL:  define.RegularPages.Settings.Path,
			Icon: "Help-Setting.svg",
			Desc: "设置 HomeLab 导航参数",
		},
	}...)

	if define.AppFlags.EnableGuide {
		apps = append(apps, model.Bookmark{
			Name: "向导页面",
			URL:  define.RegularPages.Guide.Path,
			Icon: "Help-Guide.svg",
			Desc: "页面各模块功能向导",
		})
	}

	if define.AppFlags.EnableEditor {
		apps = append(apps, model.Bookmark{
			Name: "内容编辑",
			URL:  define.RegularPages.Editor.Path,
			Icon: "Help-Editor.svg",
			Desc: "编辑导航应用、书签",
		})
	}

	apps = append(apps, []model.Bookmark{
		{
			Name: "图标挑选",
			URL:  define.RegularPages.Icons.Path,
			Icon: "Help-Mdi.svg",
			Desc: "挑选 Material Design Icons",
		},
		{
			Name: "服务导航",
			URL:  define.SettingPages.Theme.Path,
			Icon: "Help-Theme.svg",
			Desc: "HomeLab 服务架构导航",
		},
		{
			Name: "天气设置",
			URL:  define.SettingPages.Weather.Path,
			Icon: "Help-Weather.svg",
			Desc: "设定天气显示",
		},
		{
			Name: "搜索设置",
			URL:  define.SettingPages.Search.Path,
			Icon: "Help-Search.svg",
			Desc: "设置书签搜索功能",
		},
		{
			Name: "界面设置",
			URL:  define.SettingPages.Appearance.Path,
			Icon: "Help-Appearance.svg",
			Desc: "界面功能显示设置",
		},
		{
			Name: "程序版本",
			URL:  define.SettingPages.Others.Path,
			Icon: "Help-Others.svg",
			Desc: "程序介绍及程序版本信息",
		},
		{
			Name: "问题反馈",
			URL:  "https://github.com/vickai/flare/issues",
			Icon: "Help-Issues.svg",
			Desc: "GitHub Issues",
		},
	}...)

	tpl := ""

	for _, app := range apps {

		desc := ""
		if app.Desc == "" {
			desc = app.URL
		} else {
			desc = app.Desc
		}

		tpl = tpl + `
			<div class="app-container" data-id="` + app.Icon + `">
			<a href="` + app.URL + `" class="app-item" title="` + app.Name + `">
			  <div class="app-icon">` + mdi.GetIconByName(app.Icon) + `</div>
			  <div class="app-text">
				<p class="app-title">` + app.Name + `</p>
				<p class="app-desc">` + desc + `</p>
			  </div>
			</a>
			</div>
			`
	}
	return template.HTML(tpl)
}
