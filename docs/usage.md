# Usage

## Running as systemd service

Since wghttp doesn't need any privilege, it's preferred to run as systemd user service.

Copy [wghttp.service](./systemd/wghttp.service) to `~/.config/systemd/user/wghttp.service`.
After setting the environment options in `wghttp.service`, run:

```bash
systemctl --user daemon-reload
systemctl --user enable --now wghttp
```
