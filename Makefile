.PHONY: all build-c build-go clean

BINARY=bin/goblaze


C_SRC=./ccode/other.c
C_OBJ=./ccode/other.o
C_LIB=./ccode/libother.a
C_INCLUDE_DIR=./ccode/


build: build-c build-go


build-go:
	@echo "\033[0;32mBuilding Go code...\033[0m"
	
	@go build -o $(BINARY) ./cmd/main.go



build-c:
	@echo "\033[0;32mBuilding C code...\033[0m"

	@gcc -c -fPIC $(C_SRC) -I$(C_INCLUDE_DIR) -o $(C_OBJ)
	@ar rcs $(C_LIB) $(C_OBJ)
	@echo "\033[0;32mC code built successfully\033[0m"



run: build
	@./$(BINARY)


clean:
	rm -f $(BINARY) $(C_OBJ) $(C_LIB)


bench:
	@go test -benchmem  -cpuprofile cpu.prof -memprofile mem.prof -benchtime=5s   -timeout 10s -bench . -benchmem ./benchmarks 


memory:
	@go tool pprof -http=:8080  mem.prof
# install graphviz too see graphs in web

cpu:
	@go tool pprof -http=:8080  cpu.prof