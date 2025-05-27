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

Find a complete example of [network_params.yaml](https://github.com/ethpandaops/ethereum-package/blob/main/network_params.yaml) on the ethereum-package

## Standalone Setup

Clone the repo

```bash
git clone https://github.com/punkhazardlabs/txpool-viz.git && cd txpool-viz
```

Setup endpoint configs `config.yaml` from the provided template `config.example.yaml`

```yaml
endpoints:
  - name: geth
    rpc_url: 'http://127.0.0.1:57826'
    socket: 'ws://127.0.0.1:57827'
  - name: nethermind
    rpc_url: 'http://127.0.0.1:57820'
    socket: 'ws://127.0.0.1:57821'
  - name: reth
    rpc_url: 'http://127.0.0.1:57811'
    socket: 'ws://127.0.0.1:57812'
beacon_sse_url: 'http://127.0.0.1:52566' # FOCIL Enabled bean api endpoint. Leave blank if not needed
polling:
  interval: 1s
  timeout: 5s
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
