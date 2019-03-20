# Stargate

Stargate runs SOCKS proxies on different ports egressing on sequential IPs in the same subnet.
This requires the host running stargate to have the subnet routed directly to it.

## Usage

```
Usage of ./stargate:
  -listen string
        IP to listen on (default "127.0.0.1")
  -port uint
        first port to start listening on (default 10000)
  -proxy string
        CIDR notation of proxy IPs (default "127.0.0.1/32")
```

## Example

The following will start 255 SOCKS proxies listening on ports 10000-100256 sending traffic egressing on 12.34.56.0 through 12.34.56.254.
```
./stargate -listen 127.0.0.7 -port 10000 -proxy 12.34.56.0/24
```

### [Docker](https://cloud.docker.com/repository/docker/lanrat/stargate)

Stargate can be run inside Docker as well, but it will require fancy routing rules or `--net=host`.
