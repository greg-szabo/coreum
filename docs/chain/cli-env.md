# CLI environment setup

This doc describes the command to set up the environment depending on the type of network you want to use.

# Network variables

<!-- markdown-link-check-disable -->

| \-                     | znet (localnet)             | devnet                               |
|------------------------|-----------------------------|--------------------------------------|
| **Chain ID**           | coreum-devnet-1             | coreum-devnet-1                      |
| **Denom**              | udevcore                      | udevcore                               |
| **Node URL**           | http://localhost:26657      | https://s-0.devnet-1.coreum.dev:443  |
| **Faucet URL**         | http://localhost:8090       | https://api.devnet-1.coreum.dev      |
| **Cosmovisor version** | v1.3.0                      | v1.3.0                               |
| **Cored version**      | already installed via crust | check the latest devnet release      |
| **State sync servers** | not supported               | not supported                        |

<!-- markdown-link-check-enable -->

* Set the chain env variables with the "network" corresponding values.

    ```
    export CORED_CHAIN_ID="{Chain ID}"
    export CORED_DENOM="{Denom}"
    export CORED_NODE="{Node URL}"
    export CORED_FAUCET_URL="{Faucet URL}"
    export CORED_COSMOVISOR_VERSION="{Cosmovisor version}"
    export CORED_VERSION="{Cored version}"
    
    export CORED_CHAIN_ID_ARGS="--chain-id=$CORED_CHAIN_ID"
    export CORED_NODE_ARGS="--node=$CORED_NODE $CORED_CHAIN_ID_ARGS"
    
    export CORED_HOME=$HOME/.core/"$CORED_CHAIN_ID"
  
    export CORED_BINARY_NAME=$(arch | sed s/aarch64/cored-linux-arm64/ | sed s/x86_64/cored-linux-amd64/)
    export COSMOVISOR_TAR_NAME=cosmovisor-$CORED_COSMOVISOR_VERSION-linux-$(arch | sed s/aarch64/arm64/ | sed s/x86_64/amd64/).tar.gz
  
    export PATH=$PATH:$CORED_HOME/bin
    export PATH=$PATH:$CORED_HOME/cosmovisor/genesis/bin
    export DAEMON_HOME="$CORED_HOME/"
    export DAEMON_NAME="cored"
    ```

  **Attention!** *Set those variables globally to be automatically set after starting a new terminal session.*


* (Optional) set those variables globally to be automatically set after starting a new terminal session.

* (Optional) init the fund account function to use later. This function might be used later.

    ```bash
    fund_cored_account(){ 
      echo Funding account: $1
      curl --location --request POST "$CORED_FAUCET_URL/api/faucet/v1/send-money" \
    --header 'Content-Type: application/json' \
    --data-raw "{ \"address\": \"$1\"}"
    }
    ```
