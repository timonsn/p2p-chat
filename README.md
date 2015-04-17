# p2p-chat
Example Go (Golang) P2P chat application


## Example Usage
Start the p2p-chat application using:

	go run p2p.go -n "John Doe" -j 192.168.1.10:8000
	
Or build and run it:

	go build
	./p2p-chat -n "John Doe" -j 192.168.1.10:8000

	
Configuration flags:

	-n Nickname
	-p Port application listens on
	-j Single other known host:ip 
	
	
## Requirements

- Go v1.x
