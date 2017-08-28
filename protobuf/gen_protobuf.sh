# my GOPATH is more than one dir... otherwise
# set PATH=$PATH:$GOPATH/bin
DIR=`dirname $0`
set PATH=$PATH:~/go/bin
protoc --go_out=messages $DIR/*.proto
