# Rome
Ground up POC of raft concensus

### Notes
- initial peers are hard coded as http localhost ports 8000, 8001, 8002
- this has been deliberetely left very rough in order to invite instability and prove resiliency

### usage
- `go run main.go --nodeID=romulus -p 8000`
- `go run main.go --nodeID=remus -p 8001`
- `go run main.go --nodeID=wolf_bitch -p 8002`