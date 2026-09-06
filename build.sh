GOOS=linux GOARCH=amd64 go build -o pixiuctl-linux-amd64 cmd/pixiuctl.go
GOOS=linux GOARCH=arm64 go build -o pixiuctl-linux-arm64 cmd/pixiuctl.go
GOOS=windows GOARCH=amd64 go build -o pixiuctl-windows-amd64 cmd/pixiuctl.go
GOOS=windows GOARCH=arm64 go build -o pixiuctl-windows-arm64 cmd/pixiuctl.go
GOOS=darwin GOARCH=amd64 go build -o pixiuctl-darwin-amd64 cmd/pixiuctl.go
GOOS=darwin GOARCH=arm64 go build -o pixiuctl-darwin-arm64 cmd/pixiuctl.go
