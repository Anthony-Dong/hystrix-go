export GOFLAGS=


all:goget govendor



goget:
	go get -u -v go.uber.org/atomic

govendor:
	go mod vendor