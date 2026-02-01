module github.com/EasterCompany/dex-web-service

go 1.25.6

require (
	github.com/EasterCompany/dex-go-utils v0.0.0
	github.com/chromedp/chromedp v0.14.2
	github.com/redis/go-redis/v9 v9.17.3
	golang.org/x/net v0.48.0
)

replace github.com/EasterCompany/dex-go-utils => ../dex-go-utils

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chromedp/cdproto v0.0.0-20250724212937-08a3db8b4327 // indirect
	github.com/chromedp/sysutil v1.1.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-json-experiment/json v0.0.0-20250725192818-e39067aee2d2 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)
