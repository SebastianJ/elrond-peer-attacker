#!/usr/bin/env bash

echo "Compiling and downloading relevant files..."

git fetch && git pull && make clean && make spam

mkdir -p keys config data

echo "Downloading configuration files..."
curl -o config/p2p.toml https://raw.githubusercontent.com/ElrondNetwork/elrond-config/master/p2p.toml
curl -o config/economics.toml https://raw.githubusercontent.com/ElrondNetwork/elrond-config/master/economics.toml

echo "Downloading tx data files..."
curl -o data/tx_data.txt https://gist.githubusercontent.com/SebastianJ/bad62d56176a4c90201cd9e0a777624d/raw/b0bf08f6e627a2f386ee267125f386db62590da5/data.txt
curl -o data/tx_receivers.txt https://gist.githubusercontent.com/SebastianJ/3fa9e75ed6d78f85d1e73eb1370f05ad/raw/f02dfc2e4e7d19f25cfa8a2fe3390aa3f2004b34/receivers.txt

echo "Downloading smart contract data files..."
curl -o data/sc_data.txt https://gist.githubusercontent.com/SebastianJ/fe7b66029391742d25c4dfaa0a1ac053/raw/542a04763f8c2b05aa9aa1d5441291c1285d8276/data.txt
curl -o data/sc_receivers.txt https://gist.githubusercontent.com/SebastianJ/d127e1fb78ed0ed74d1a33ca52f62775/raw/02b5103980ce05c28d96a088d92dd3629d65dc2e/receivers.txt
