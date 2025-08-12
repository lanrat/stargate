# Stargate

Stargate runs TCP SOCKS proxies on different ports egressing on sequential IPs in the same subnet.
This requires the host running stargate to have the subnet routed directly to it.

If you have an IPv6 subnet, stargate can allow you to make full use of it by any program that can speak SOCKS.

## Usage

```console
Usage of ./stargate: [OPTION]... CIDR
        CIDR example: "192.0.2.0/24"
OPTIONS:
  -listen string
        IP to listen on (default "localhost")
  -port uint
        first port to start listening on
  -random uint
        port to use for random proxy server
  -verbose
        enable verbose logging
  -version
        print version and exit
```

## Random

The `-random` flag starts a SOCKS5 proxy that egresses traffic on a random IP in the subnet.
This is useful to avoid rate-limiting or in situations where there are too many IPs in the subnet to listen on each port which is common with IPv6.

## Example

The following will start 254 SOCKS proxies listening on 127.0.0.7 ports 10001-100254 sending traffic egressing on 192.0.2.1 through 192.0.2.254.

```console
./stargate -listen 127.0.0.7 -port 10001 192.0.2.0/24
```

The following will start a single socks proxy listening on 127.0.0.1:1337 egressing each connection from a random IP in 2001:DB8:1337::1/64 This offers you 2<sup>64</sup> possible IPs to egress on.

```console
./stargate -random 1337 2001:DB8:1337::1/64

```

## Download

### [Precompiled Binaries](https://github.com/lanrat/stargate/releases)


### [Docker](https://github.com/lanrat/stargate/pkgs/container/stargate)

Stargate can be run inside Docker as well, but it will require fancy routing rules or `--net=host`.

```shell
docker pull ghcr.io/lanrat/stargate:latest
```

## Building

Just run `make`!
