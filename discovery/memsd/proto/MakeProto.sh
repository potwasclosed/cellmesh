#!/usr/bin/env bash

CURRDIR=`pwd`
cd ../../../
export WORKPATH=`pwd`

set -e
Protoc=${WORKPATH}/bin/protoc

# cellmesh服务绑定
CellMeshProtoGen=${WORKPATH}/bin/cmprotogen
PkgPath=${WORKPATH}/pkg
echo  111
echo  ${PkgPath}
go build  -v  -o ${CellMeshProtoGen}  github.com/davyxu/cellmesh/tool/protogen

echo ${CellMeshProtoGen}
# 协议生成
ProtoPlusGen=${WORKPATH}/bin/protoplus
go build -v -o ${ProtoPlusGen} github.com/davyxu/protoplus

cd ${CURRDIR}

# windows下时，添加后缀名
if [ `go env GOHOSTOS` == "windows" ];then
	EXESUFFIX=.exe
fi

echo "生成服务器协议的go消息..."
${ProtoPlusGen} -package=proto -go_out=msgsvc_gen.go `source ./protolist.sh svc`

echo "生成服务器协议的消息绑定..."
${CellMeshProtoGen} -package=proto -cmgo_out=msgbind_gen.go `source ./protolist.sh svc`