.PHONY: build build-linux build-mac

buildtime_windows= $(shell echo %date:~0,4%%date:~5,2%%date:~8,2% %time:~0,2%:%time:~3,2%:%time:~6,2%)
buildtime_linux= $(shell date +"%Y-%M-%d %H:%M:%S")
version = 0.2.8
ldflags_windows = "-s -w -X 'mitmgo/src/manage.Version=${version}' -X 'mitmgo/src/manage.BuildTime=${buildtime_windows}'"
ldflags_linux = "-s -w -X 'mitmgo/src/manage.Version=${version}' -X 'mitmgo/src/manage.BuildTime=${buildtime_linux}'"
ldflags_mac = "-s -w -X 'mitmgo/src/manage.Version=${version}' -X 'mitmgo/src/manage.BuildTime=${buildtime_linux}'"

build:
	go build -ldflags ${ldflags_windows} -o ./release/mitmgo.exe
build-linux:
	go env -w GOOS=linux
	go build -ldflags ${ldflags_linux} -o ./release/mitmgo
build-mac:
	go env -w GOOS=darwin
	go build -ldflags ${ldflags_mac} -o ./release/mitmgo_mac
