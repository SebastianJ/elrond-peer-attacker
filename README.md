# elrond-peer-attacker

Misc p2p attack tools targeting Elrond's blockchain

## Eclipse

To build the binary: `cd cmd/eclipse && go build`

Start 500 seed nodes: `./eclipse --count 500`

## SeedNodeDDoS

To build the binary: `cd cmd/seednodeddos && go build`

Start 500 DDoS pingers: `./seednodeddos --count 500`

## P2P Spammer:
```
./dist/spam --count 50 --config config/p2p.toml --economics-config config/economics.toml --data data/data.txt --receivers data/receivers.txt 
```
