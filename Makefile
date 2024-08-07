build: build-go build-c

build-go:
	@go build -o bin/goblaze

build-c:
	@gcc -I ./c_code/include -o c_code/c_code ./c_code/src/*.c

run: build-go
	@./bin/goblaze
