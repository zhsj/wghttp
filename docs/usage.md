# Usage

## Running as systemd service

Since wghttp doesn't need any privilege, it's preferred to run as systemd user service.

Copy [wghttp.service](./systemd/wghttp.service) to `~/.config/systemd/user/wghttp.service`.
After setting the environment options in `wghttp.service`, run:

```bash
systemctl --user daemon-reload
systemctl --user enable --now wghttp
```

## Options compared to WireGuard configuration file

For connecting as a client to a VPN gateway, you might have:

```ini
[Interface]
Address = 10.200.100.8/24
DNS = 10.200.100.1
PrivateKey = oK56DE9Ue9zK76rAc8pBl6opph+1v36lm7cXXsQKrQM=

[Peer]
PublicKey = GtL7fZc/bLnqZldpVofMCD6hDjrK28SsdLxevJ+qtKU=
AllowedIPs = 0.0.0.0/0
Endpoint = demo.wireguard.com:51820
PresharedKey = /UwcSPg38hW/D9Y3tcS1FOV0K1wuURMbS0sesJEP5ak=
```

The above configuration is equal to:

```bash
wghttp \
  --client-ip=10.200.100.8 \
  --dns=10.200.100.1 \
  --private-key=oK56DE9Ue9zK76rAc8pBl6opph+1v36lm7cXXsQKrQM= \
  --peer-key=GtL7fZc/bLnqZldpVofMCD6hDjrK28SsdLxevJ+qtKU= \
  --peer-endpoint=demo.wireguard.com:51820 \
  --preshared-key=/UwcSPg38hW/D9Y3tcS1FOV0K1wuURMbS0sesJEP5ak= \
  --exit-mode=remote
```

## Dynamic DNS

When your server IP is not persistent, you can set a domain with
DDNS for it. `wghttp` will resolve the domain periodically.

- `--resolve-dns=`

  By default, the server domain is resolved by system resolver.
  This option can be set to use a different DNS server.

- `--resolve-interval=`

  This option controls the interval for resolving server domain.

Set `--resolve-interval=` to `0` to disable this behaviour.

## DNS server format

Both `--dns=` and `--resolve-dns=` options support following format:

- Plain DNS

  `8.8.8.8`, `udp://8.8.8.8`, `tcp://8.8.8.8`,
  `8.8.8.8:53`, `udp://8.8.8.8:53`, `tcp://8.8.8.8:53`,

- DNS over TLS

  `tls://8.8.8.8`, `tls://8.8.8.8:853`

- DNS over HTTPS

  `https://8.8.8.8`
