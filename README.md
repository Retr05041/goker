# Goker
(Mental Poker Protocol)[https://en.wikipedia.org/wiki/Mental_poker] with Time Locked Keys.

A program created for my final year dissertation.

# What is it?
Goker is a decentralized peer-to-peer poker client, implementing the Mental Poker Protocol to establish trust through cummutative encryption. Signing all commands with an RSA signature, and tags game commands to prevent core replay attacks.

# Setup and Run
To build you will need the latest version of [Golang](https://go.dev/) installed.

If you have Make, simply run `make` and the executable will be found in `bin/Goker`

If you do not have Make, run `go build -o ./bin/Goker .`