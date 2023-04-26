# data-proxy

### Intriduction
This is a small data extract and transform program written in go. The original intention of this project is to develop pipeline to dump victoriametrics (vm for short) time-series data into downstream middleware, such as kafka and pulsar, etc. As vm still doesn't provide kafka integration for free use (only for enterprise version), I decide to open-source this program for anyone in need. 

The program contains three parts:
  - source: where is the data from
    - kafka
    - pulsar
    - prometheus remote-write api
    - ... (you can implement one yourself)
  - processor: how to process the data
    - empty (nothing to do)
    - label filter
    - ... (you can implement one yourself)
  - sink: where to write data to
    - victoriametrics
    - kafka
    - pulsar
    - ... (you can implement one yourself)

### Installation

- prepare (must): 
  - golang environment, prefer version >=1.20
- prepare (optional):
  - docker-ce with docker-compose, prefer version >= 20.10
  - make
- build:
  1. use make file:
  ```shell
  make clean build
  ```
  2. use go cli:
  ```shell
  go mod tidy
  CGO_ENABLED=0 go build
  ```
- configure:
  see details in conf-etl.yml or conf-vmagent-sidecar.yml
- run:
  1. run binary:
  ```shell
  ./data-proxy -c conf-vmagent-sidecar.yml
  ```
  2. run container
  ```shell
  docker-compose -f docker-compose.yml up -d
  ```
  