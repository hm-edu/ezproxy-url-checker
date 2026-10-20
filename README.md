# ezproxy-url-checker

Check whether a URL exists in your EZproxy config.

## Quickstart

Download the binary from [the latest release](https://github.com/hm-edu/ezproxy-url-checker/releases/latest). An automated way to do that might be:

```
./scripts/download-binary.sh
```

Generate the files the service needs from your EZProxy config and start the service

```
./ezproxy-url-checker generate /path/to/ezproxy/config.txt/folder
./ezproxy-url-checker
```

`generate` reads `config.txt` and every file it pulls in via `IncludeFile`, and writes
the `URL`, `Host`/`HJ` and `Domain` entries to `config/urls.txt`, `config/hosts.txt`
and `config/domains.txt` (use `-out DIR` to write them somewhere else).

See if google scholar is in your EZproxy config

```
curl "http://localhost:8888/proxyUrl?url=https://scholar.google.com"
```


## Deployment

**Coming soon**: instructions on how to get the config files needed for the service from EZProxy onto your system.

### init

To deploy this service to a server, first, you'll need to deploy the binary to your system.

```
$ cd /opt
$ git clone https://github.com/hm-edu/ezproxy-url-checker.git
$ cd ezproxy-url-checker
$ ./scripts/download-binary.sh
```

Then you'll need to start the service with your preferred init system.

#### systemd

If you use systemd, you could create a unit file for this service. 

```
$ cat << EOF >/etc/systemd/system/ezproxy-url-checker.service
[Install]
WantedBy=multi-user.target

[Unit]
Description=ezproxy-url-checker
After=network.target

[Service]
WorkingDirectory=/opt/ezproxy-url-checker
Restart=on-failure
RestartSec=30s
ExecStart=/opt/ezproxy-url-checker/ezproxy-url-checker
EOF

$ systemctl enable ezproxy-url-checker.service
$ systemctl start ezproxy-url-checker.service
```

### docker

Each release `vX.Y.Z` is published to GHCR as `vX.Y.Z`, `X.Y.Z`, `X.Y`, `X` and
`latest`; every push to `main` is published as `main`. Replace `<release-tag>` below
with one of those tags.

The image contains only the static binary, so use its `generate` subcommand to
create the config files from your EZProxy directory, then mount them read-only when
starting the service:

```sh
mkdir -p config
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v /path/to/ezproxy/config.txt/folder:/ezproxy:ro \
  -v "$PWD/config:/app/config" \
  ghcr.io/hm-edu/ezproxy-url-checker:<release-tag> generate /ezproxy

docker run --rm \
  --read-only \
  --cap-drop=ALL \
  --security-opt=no-new-privileges \
  -p 8888:8888 \
  -v "$PWD/config:/app/config:ro" \
  ghcr.io/hm-edu/ezproxy-url-checker:<release-tag>
```

The container runs as UID `1000` and listens on port `8888` by default. The config
files must be readable by that user; set `LISTEN` to change the address and port.

## TODO

- [x] Integration tests
- [ ] CI/CD instructions
- [ ] Better API Documentation
