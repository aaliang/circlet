# my GOPATH is more than one dir... otherwise
# set PATH=$PATH:$GOPATH/bin
set PATH=$PATH:~/go/bin
protoc --go_out=../messages *.proto
