package model

import (
	"time"
)

// Generic Bookmark Data Model
type Bookmark struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"link"`
	Icon     string `yaml:"icon,omitempty"`
	Desc     string `yaml:"desc,omitempty"`
	Private  bool   `yaml:"private,omitempty"`
	Category string `yaml:"category,omitempty"`
}

// Generic Category Data Model
type Category struct {
	ID   string `yaml:"id"`
	Name string `yaml:"title"`
}

// Generic Bookmarks Data Model
type Bookmarks struct {
	Categories []Category `yaml:"categories,omitempty"`
	Items      []Bookmark `yaml:"links"`
}

// 支持 IP 探测
type VickaiBookmark struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"link"`
	Icon     string `yaml:"icon,omitempty"`
	Desc     string `yaml:"desc,omitempty"`
	IP   	 string `yaml:"ip,omitempty"`
	Port     int    `yaml:"port,omitempty"`
}

// VickaiServiceGroup 用于支持分类
type VickaiServiceGroup struct {
	Category string           `yaml:"category"`
	Items    []VickaiService  `yaml:"items"`
}

// 支持 IP 端口服务在线探测
type VickaiService struct {
	Name     string `yaml:"name"`
	IP   	 string `yaml:"ip,omitempty"`
	Port     int    `yaml:"port,omitempty"`
}

// TailscaleStatus 用于解析 tailscale status --json
type TailscaleStatus struct {
	Self struct {
		HostName     string    `json:"HostName"`
		DNSName      string    `json:"DNSName"`
		Online       bool      `json:"Online"`
		TailscaleIPs []string  `json:"TailscaleIPs"`
		OS           string    `json:"OS"`
		Relay        string    `json:"Relay"`
		CurAddr      string    `json:"CurAddr"`
		LastSeen     time.Time `json:"LastSeen"`
	} `json:"Self"`
	Peer map[string]struct {
		HostName     string    `json:"HostName"`
		DNSName      string    `json:"DNSName"`
		Online       bool      `json:"Online"`
		TailscaleIPs []string  `json:"TailscaleIPs"`
		OS           string    `json:"OS"`
		Relay        string    `json:"Relay"`
		CurAddr      string    `json:"CurAddr"`
		LastSeen     time.Time `json:"LastSeen"`
	} `json:"Peer"`
}

