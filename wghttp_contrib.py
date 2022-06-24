#!/usr/bin/env python3

import argparse
import re
from pathlib import Path

ipRx = re.compile("User contributions for ((?:\\d+\\.){3}\\d+)")

PORT = 25344


p = argparse.ArgumentParser(description="wghttp starter script")
sp = p.add_subparsers(dest="cmd")
sp.add_parser("ip")

rp = sp.add_parser("run")
rp.add_argument("cfg", help="Path to config file", type=Path)

rp = sp.add_parser("emit")
rp.add_argument("cfg", help="Path to config file", type=Path)


def getIP(port: int) -> str:
	try:
		import httpx as http
	except ImportError:
		import requests as http

	return ipRx.search(
		http.get(
			"https://en.wikipedia.org/wiki/Special:MyContributions",
			proxies={
				"https://": "socks5://localhost:" + str(port),
				"http://": "socks5://localhost:" + str(port),
			},
		).text
	).group(1)


def envDictFromFile(cfgPath: Path):
	import ipaddress

	import wg_conf

	cfg = wg_conf.WireguardConfig(cfgPath)

	d = {}
	d["DNS"] = "8.8.8.8"
	d["LISTEN"] = "localhost:" + str(PORT)
	d["EXIT_MODE"] = "remote"  # local

	d["PRIVATE_KEY"] = cfg.interface["PrivateKey"]
	d["CLIENT_IP"] = str(ipaddress.ip_network(cfg.interface["Address"]).network_address)

	p = next(iter(cfg.peers.values()))
	d["PEER_KEY"] = p["PublicKey"]
	d["PEER_ENDPOINT"] = p["Endpoint"]

	return d


RUN_COMMAND = ("./wghttp", "-v")


def startProxy(config: Path, port: int) -> int:
	import os
	import subprocess

	os.environ.update(envDictFromFile(config))

	return subprocess.call(RUN_COMMAND)


def emitBashScript(config: Path, port: int):
	print("#!/usr/bin/env bash")
	for k, v in envDictFromFile(config).items():
		print("export", k + "=" + v, ";")

	print(" ".join(RUN_COMMAND))


if __name__ == "__main__":
	args = p.parse_args()
	if args.cmd == "ip":
		print("My IP:", getIP(PORT))
		exit(0)
	elif args.cmd == "run":
		exit(startProxy(args.cfg, PORT))
	elif args.cmd == "emit":
		emitBashScript(args.cfg, PORT)
	else:
		p.print_help()
		exit(1)
