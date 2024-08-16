.PHONY: all build-c build-go clean

BINARY=bin/goblaze

C_SRC=$(wildcard ./ccode/*.c)
C_OBJ=$(C_SRC:.c=.o)
C_LIB=./ccode/libother.a
C_INCLUDE_DIR=./ccode/

build: build-c build-go

build-go:
	@echo -e "\033[0;32mBuilding Go code...\033[0m"
	@go build -o $(BINARY) ./cmd/main.go

build-c: $(C_OBJ)
	@echo -e "\033[0;32mBuilding C code...\033[0m"
	@ar rcs $(C_LIB) $(C_OBJ)
	@echo -e "\033[0;32mC code built successfully\033[0m"

%.o: %.c
	@echo "Compiling $< to $@..."
	@gcc -c -fPIC $< -I$(C_INCLUDE_DIR) -o $@

run: build
	@./$(BINARY)

clean:
	rm -f $(BINARY) $(C_OBJ) $(C_LIB)

bench:
	@go test  -benchmem   -cpuprofile=cpu.prof -memprofile=mem.prof -benchtime=10s -timeout 320s -bench . -benchmem ./benchmarks -count 3
	

benchGC:
	@go test -gcflags="-e" -benchmem   -cpuprofile cpu.prof -memprofile mem.prof -benchtime=10s -timeout 320s -bench . -benchmem ./benchmarks -count 3






benchfast:
	@echo "\033[0;32mRunning fasthttp tests...\033[0m"
	@for i in 1 2 3; do \
		echo  "\033[0;33mRun $$i for fasthttp  tests\033[0m"; \
		go run ./benchwithout/fasthttptest/main.go; \
	done

benchdefault:
	@echo "\033[0;32mRunning fasthttp default tests...\033[0m"
	@for i in 1 2 3; do \
		echo  "\033[0;33mRun $$i for fasthttp default tests\033[0m"; \
		go run ./benchwithout/fasthttpdefault/main.go; \
	done

benchgoblaze:
	@echo "\033[0;32mBuilding goblaze tests...\033[0m"
	@for i in 1 2 3; do \
		echo  "\033[0;33mRun $$i for goblaze \033[0m"; \
		go run ./benchwithout/goblazetest/main.go; \
	done

run_all: benchdefault benchfast benchgoblaze
	@echo "\033[0;31mAll benchmarks completed...\033[0m"

memory:
	@go tool pprof -http=:8080 mem.prof 
# install graphviz too see graphs in web

cpu:
	@go tool pprof -http=:8080 cpu.prof

