#/bin/bash

docker stop data-proxy
docker rm data-proxy

root=`pwd`

docker run --name data-proxy -m 512m -p 9423:9423 -v ${root}/conf-vmagent-sidecar.yml:/etc/conf-vmagent-sidecar.yml data-proxy:latest -c /etc/conf-vmagent-sidecar.yml
