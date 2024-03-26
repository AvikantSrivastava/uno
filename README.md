# UNO Multiplayer 

![UNO_Logo](https://github.com/mahimdashora/UNO-game/assets/60029463/4198de68-0a20-44ac-81a1-3cd0a459d29a)
# Uno Game Setup Guide

This guide provides step-by-step instructions to set up and run the Uno game.

## Step 1: Define Player Names

Open the `main.go` file and define player names in the `players` slice.

```go
// Example:
// players := []string{"Player1", "Player2", "Player3"}
```
## Step 2: Run the Uno Game

1. Navigate to the `cmd/uno` directory.
2. Run the `main.go` file.

```bash
cd cmd/uno
go run main.go
```
## Step 3: Connect to the Server
### Windows:

To connect to the server port, use `wscat` with the following command:

```bash
wscat -c ws://localhost:8080/ws
```
- Replace `8080` with your specific port number if needed.
- Enter the player name correctly when prompted.
- Wait for your turn to play.
