.PHONY: all build-c build-go clean

BINARY=bin/goblaze

C_SRC=$(wildcard ./ccode/*.c)
C_OBJ=$(C_SRC:.c=.o)
C_LIB=./ccode/libother.a
C_INCLUDE_DIR=./ccode/

build: build-c build-go

build-go:
	@echo "\033[0;32mBuilding Go code...\033[0m"
	@go build -o $(BINARY) ./cmd/main.go

build-c: $(C_OBJ)
	@echo "\033[0;32mBuilding C code...\033[0m"
	@ar rcs $(C_LIB) $(C_OBJ)
	@echo "\033[0;32mC code built successfully\033[0m"

%.o: %.c
	@echo "Compiling $< to $@..."
	@gcc -c -fPIC $< -I$(C_INCLUDE_DIR) -o $@

run: build
	@./$(BINARY)

clean:
	rm -f $(BINARY) $(C_OBJ) $(C_LIB)

bench:
	@go test -benchmem -cpuprofile cpu.prof -memprofile mem.prof -benchtime=5s -timeout 20s -bench . -benchmem ./benchmarks -count 3

memory:
	@go tool pprof -http=:8080 mem.prof # install graphviz too see graphs in web

cpu:
	@go tool pprof -http=:8080 cpu.prof

