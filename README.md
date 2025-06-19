# Mempool Visualizer

This project is still in development. This repo represents the core txpool-viz logic

[![Docker Build and Publish](https://github.com/punkhazardlabs/txpool-viz/actions/workflows/docker.yml/badge.svg)](https://github.com/punkhazardlabs/txpool-viz/actions/workflows/docker.yml)

[Original Idea Proposal](https://github.com/ethpandaops/tooling-wishlist/blob/master/open-ideas/txpool-viz.md)

[Project Plan](https://github.com/punkhazardlabs/txpool-viz/wiki)

[Discord Server](https://discord.gg/kq6YXwCzDs)

## Introduction:
A tool to visualize and compare transaction pools across different Ethereum execution clients. This helps debug transaction propagation issues, analyze mempool behaviors during attacks, and understand differences in transaction acceptance policies between clients.

## Use with Kurtosis

This is currently in development mode. Please open an issue in this repo to report any bugs

Install [Kurtosis](https://docs.kurtosis.com/)

This tool has been added as an additional service on the [ethereum-package](https://github.com/ethpandaops/ethereum-package) for Kurtosis

Using `network_params.yaml` from this repo, run our ethereum-package fork (for development)

```bash
kurtosis run --enclave my-testnet github.com/punkhazardlabs/ethereum-package@ndeto/feat/add-txpool-viz --args-file network_params.yaml
```

You will see an output similar to the one below. Visit the txpool-viz url

![Sample Output](doc/output.png)

Find a complete spec of [network_params.yaml](https://github.com/ethpandaops/ethereum-package?tab=readme-ov-file#configuration) config on the ethereum-package

### [FOCIL](https://eips.ethereum.org/EIPS/eip-7805) - Visualization

Use the `focil_params.yaml` config file and run with Kurtosis

```bash
kurtosis run --enclave my-testnet github.com/punkhazardlabs/ethereum-package@ndeto/feat/add-txpool-viz --args-file focil_params.yaml
```

Visit the web UI to see the visualization.

## Standalone Setup

Clone the repo

```bash
git clone https://github.com/punkhazardlabs/txpool-viz.git && cd txpool-viz
```

Setup endpoint configs `cfg/config.yaml` from the provided template `cfg/config.example.yaml`

```yaml
endpoints:
  - name: reth-prysm
    rpc_url: "http://127.0.0.1:55405"
    socket: "ws://127.0.0.1:55401"
  - name: geth-lodestar
    rpc_url: "http://127.0.0.1:55398"
    socket: "ws://127.0.0.1:55399"
  - name: nethermind-teku
    rpc_url: "http://127.0.0.1:55393"
    socket: "ws://127.0.0.1:55394"
polling:
  interval: 0.1s
  timeout: 5s
filters:
  min_gas_price: 1gwei
log_level: "info"
focil_enabled: "false"  # Only use if your network is FOCIL enabled (https://eips.ethereum.org/EIPS/eip-7805)
beacon_urls: # Only used if focil_enabled = "true". 
  - name: reth-prysm
    beacon_url: "http://127.0.0.1:55410"
  - name: geth-lodestar
    beacon_url: "http://127.0.0.1:55426"
  - name: nethermind-teku 
    beacon_url: "http://127.0.0.1:55652"
```

Setup storage instances by running the docker-compose file

```conf
docker compose -f 'docker-compose.yml' up -d --build 
```

Set up .env file from `.env.example.` This is for running this repo only, these variables will be setup automatically when in use with Kurtosis


```conf
POSTGRES_URL=postgres://postgres:password@localhost:5432/postgres?sslmode=disable
REDIS_URL=redis://localhost:6379/0
PORT=42069
```

Run the tool

```bash
make run
```

The web ui will be served on port 42069

```
http://localhost:42069
```

Local Development Tools:
- [Kurtosis Ethereum Package](https://github.com/ethpandaops/ethereum-package) - Simulate a local testnet
- [Spamoor](https://github.com/ethpandaops/spamoor) - Send spam tx's to your local testnet mempool
- `docker-compose.yml` is provided to setup local dependencies
