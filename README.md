[Discord Server](https://discord.gg/UEyf7U8Smr)

# Mempool Visualizer

[Original Idea Proposal](https://github.com/ethpandaops/tooling-wishlist/blob/master/open-ideas/txpool-viz.md)

## Introduction:
A tool to visualize and compare transaction pools across different Ethereum execution clients. This helps debug transaction propagation issues, analyze mempool behaviors during attacks, and understand differences in transaction acceptance policies between clients.

## Set up

Install dependencies

```bash
make tidy
```

Setup endpoint configs using `config.example.yaml`

Set up .env file from .env.example. This is for running this repo only, these variables will be setup automatically when in use with Kurtosis

```
POSTGRES_URL=postgres://postgres:password@localhost:5432/postgres?sslmode=disable
REDIS_URL=redis://localhost:6379/0
```

Run the code

```bash
make run
```
