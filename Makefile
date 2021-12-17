all: build build-linux build-mac
.PHONY: build build-linux build-mac

buildtime_windows= $(shell echo %date:~0,4%%date:~5,2%%date:~8,2% %time:~0,2%:%time:~3,2%:%time:~6,2%)
buildtime_linux=
version = 0.2.8
ldflags_windows = "-s -w -X 'mitmgo/src/manage.Version=${version}' -X 'mitmgo/src/manage.BuildTime=${buildtime_windows}'"
ldflags_linux = "-s -w -X 'mitmgo/src/manage.Version=${version}' -X 'mitmgo/src/manage.BuildTime=${buildtime_linux}'"
ldflags_mac = "-s -w -X 'mitmgo/src/manage.Version=${version}' -X 'mitmgo/src/manage.BuildTime=${buildtime}'"

pes_parent_dir:=$(shell pwd)/$(lastword $(MAKEFILE_LIST))
pes_parent_dir:=$(shell dirname $(pes_parent_dir))


build:
	go build -ldflags ${ldflags_windows} -o ./release/passivescanner.exe
build-linux:
	go env -w GOOS=linux
	go build -ldflags ${ldflags_linux} -o ./release/passivescanner
build-mac:
	go env -w GOOS=darwin
	go build -ldflags ${ldflags_mac} -o ./release/passivescanner_mac
