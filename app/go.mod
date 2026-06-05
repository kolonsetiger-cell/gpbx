module gitee.com/kolonse_zhjsh/gpbx/app

replace gitee.com/kolonse_zhjsh/gpbx/kcfg => ../kcfg

replace gitee.com/kolonse_zhjsh/gpbx/log => ../log

replace gitee.com/kolonse_zhjsh/gpbx/esl => ../esl

replace gitee.com/kolonse_zhjsh/gpbx/store => ../store

go 1.25.6

require (
	gitee.com/kolonse_zhjsh/gpbx/kcfg v0.0.0-00010101000000-000000000000
	gitee.com/kolonse_zhjsh/gpbx/log v0.0.0-20151011154852-f2fc43724d8d
	gitee.com/kolonse_zhjsh/gpbx/store v0.0.0-00010101000000-000000000000
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/glebarez/sqlite v1.11.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/kolonse/logs v0.0.0-20151011154852-f2fc43724d8d // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	gorm.io/gorm v1.31.1 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
)
