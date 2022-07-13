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
```

The above configuration is equal to:

```bash
wghttp \
  --peer-endpoint=demo.wireguard.com:51820 \
  --peer-key=GtL7fZc/bLnqZldpVofMCD6hDjrK28SsdLxevJ+qtKU= \
  --private-key=oK56DE9Ue9zK76rAc8pBl6opph+1v36lm7cXXsQKrQM= \
  --client-ip=10.200.100.8 \
  --dns=10.200.100.1 \
  --exit-mode=remote
```
