# Mempool Visualizer

This project is still in development. This repo represents the core txpool-viz logic

[Original Idea Proposal](https://github.com/ethpandaops/tooling-wishlist/blob/master/open-ideas/txpool-viz.md)

[Project Plan](https://github.com/punkhazardlabs/txpool-viz/wiki)

[Discord Server](https://discord.gg/kq6YXwCzDs)

## Introduction:
A tool to visualize and compare transaction pools across different Ethereum execution clients. This helps debug transaction propagation issues, analyze mempool behaviors during attacks, and understand differences in transaction acceptance policies between clients.

## Use with Kurtosis

This is currently in development mode. Please open an issue in this repo to report any bugs

Install [Kurtosis](https://docs.kurtosis.tech)

This tool has been added as an additional service on the [ethereum-package](https://github.com/ethpandaops/ethereum-package) for Kurtosis

Create the local config file `network_params.yaml`, and copy the following config.

```yaml
participants:
  - el_type: geth
    el_image: ethereum/client-go:latest
    cl_type: lighthouse
    cl_image: sigp/lighthouse:latest-unstable
    vc_type: lighthouse
    vc_image: sigp/lighthouse:latest-unstable
    count: 3
network_params:
  genesis_delay: 20
  electra_fork_epoch: 0
  eip7805_fork_epoch: 1
  seconds_per_slot: 6
  num_validator_keys_per_node: 256
additional_services:
  - txpool_viz                      # Add txpool_viz as an additional service
  - spamoor
port_publisher:
  additional_services:
    enabled: true
    public_port_start: 65500
spamoor_params:
  spammers:
    - scenario: eoatx
      config:
        throughput: 10
    - scenario: uniswap-swaps
      config:
        throughput: 10
    - scenario: blob-combined
      config:
        throughput: 5
txpool_viz_params:                 # Optional txpool_viz params
  polling:
    interval: 1s
    timeout: 5s
  filters:
    min_gas_price:
  focil_enabled: "true"

```

Find a complete example of [network_params.yaml](https://github.com/ethpandaops/ethereum-package/blob/main/network_params.yaml) on the ethereum-package

Run the ethereum package

```bash
# Currently running with a fork for development
kurtosis run --enclave my-testnet github.com/punkhazardlabs/ethereum-package@ndeto/feat/add-txpool-viz --args-file network_params.yaml
```

You will see an output similar to the one below. Visit the txpool-viz url

![Sample Output](doc/output.png)


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
