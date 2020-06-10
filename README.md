# elrond-peer-attacker

This repo consists of two components: eclipse and seednodeddos.

Eclipse launches a specified amount of seed nodes that essentially only propagate fake peer connections.

SeedNodeDDoS is a DDoS tool for mass pinging a specific seed node to deny peers from connecting to it.

## Eclipse

To build the binary: `cd cmd/eclipse && go build`

Start 500 seed nodes: `./eclipse --count 500`

## SeedNodeDDoS

To build the binary: `cd cmd/seednodeddos && go build`

Start 500 DDoS pingers: `./seednodeddos --count 500`

## P2P Spammer:
```
./dist/spam --count 50 --config config/p2p.toml --economics-config config/economics.toml --data data/data.txt
```
