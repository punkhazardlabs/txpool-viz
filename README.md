# Mempool Visualizer

This project is still in development. This repo represents the core txpool-viz logic

[Original Idea Proposal](https://github.com/ethpandaops/tooling-wishlist/blob/master/open-ideas/txpool-viz.md)

[Project Plan](https://github.com/punkhazardlabs/txpool-viz/wiki)

[Discord Server](https://discord.gg/UEyf7U8Smr)

## Introduction:
A tool to visualize and compare transaction pools across different Ethereum execution clients. This helps debug transaction propagation issues, analyze mempool behaviors during attacks, and understand differences in transaction acceptance policies between clients.

## Set up

Install dependencies

```bash
make tidy
```

Setup endpoint configs using `config.example.yaml`

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

Set up .env file from .env.example. This is for running this repo only, these variables will be setup automatically when in use with Kurtosis

```
POSTGRES_URL=postgres://postgres:password@localhost:5432/postgres?sslmode=disable
REDIS_URL=redis://localhost:6379/0
PORT=42069
```

Run the code

```bash
make run
```

The web ui will be served on port 8080

```
http://localhost:8080
```

Local Development Tools:
- [Kurtosis Ethereum Package](https://github.com/ethpandaops/ethereum-package) - Simulate a local testnet
- [Spamoor](https://github.com/ethpandaops/spamoor) - Send spam tx's to your local testnet mempool
- `docker-compose.yml` is provided to setup local dependencies
