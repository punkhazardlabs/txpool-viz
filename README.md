# Mempool Visualizer

This project is still in development. This repo represents the core txpool-viz logic

[Original Idea Proposal](https://github.com/ethpandaops/tooling-wishlist/blob/master/open-ideas/txpool-viz.md)

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
  - name: nethermind
    rpc_url: 'http://127.0.0.1:62956'
    socket: 'ws://127.0.0.1:62957'
  - name: geth
    rpc_url: 'http://127.0.0.1:62945'
    socket: 'ws://127.0.0.1:62946'
  - name: reth
    rpc_url: 'http://127.0.0.1:63146'
    socket: 'ws://127.0.0.1:63147'
polling:
  interval: 5s
  timeout: 5s
filters:
  min_gas_price: 1gwei
```

Set up .env file from .env.example. This is for running this repo only, these variables will be setup automatically when in use with Kurtosis

```
POSTGRES_URL=postgres://postgres:password@localhost:5432/postgres?sslmode=disable
REDIS_URL=redis://localhost:6379/0
```

Run the code

```bash
make run
```

Local Development Tools:
- [Kurtosis Ethereum Package](https://github.com/ethpandaops/ethereum-package) - Simulate a local testnet
- [Spamoor](https://github.com/ethpandaops/spamoor) - Send spam tx's to your local testnet mempool
- `docker-compose.yml` is provided to setup local dependencies
