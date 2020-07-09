# Stargate

Stargate runs TCP SOCKS proxies on different ports egressing on sequential IPs in the same subnet.
This requires the host running stargate to have the subnet routed directly to it.

## Usage

```console
Usage of ./stargate:
  -listen string
        IP to listen on (default "127.0.0.1")
  -port uint
        first port to start listening on
  -proxy string
        CIDR notation of proxy IPs (default "127.0.0.1/32")
  -random uint
        port to use for random proxy server
```

## Random

The `-random` flag starts a SOCKS5 proxy that egresses traffic on a random IP in the subnet.
This is useful for avoid rate-limiting or in situations where there are too many IPs in the subnet to listen on each port which is common with IPv6.

## Example

The following will start 255 SOCKS proxies listening on 127.0.0.7 ports 10000-100256 sending traffic egressing on 12.34.56.0 through 12.34.56.254.

```console
./stargate -listen 127.0.0.7 -port 10000 -proxy 12.34.56.0/24
```

The following will start a single socks proxy listening on 127.0.0.1:1337 egressing each connection from a random IP in 12.34.56.0/24.

```console
./stargate -random 1337 -proxy 12.34.56.0/24

```

### [Docker](https://cloud.docker.com/repository/docker/lanrat/stargate)

Stargate can be run inside Docker as well, but it will require fancy routing rules or `--net=host`.

## Building

Just run `make`!
