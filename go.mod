module github.com/digtux/laminar

go 1.15

require (
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/aws/aws-sdk-go v1.31.11
	github.com/go-git/go-git/v5 v5.3.0
	github.com/gobwas/glob v0.2.3
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/kevinburke/ssh_config v1.1.0 // indirect
	github.com/labstack/echo/v4 v4.6.3 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/tidwall/buntdb v1.1.2
	go.uber.org/zap v1.15.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
)

// replace github.com/go-git/go-git/v5 => ../../zeripath/go-git
