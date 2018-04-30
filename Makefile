DATA_DIR=data
SRC_FILES=bindata.go $(filter-out $(wildcard *_test.go), $(wildcard *.go))

all: build

build: $(GOPATH)/bin/octocatmd

.PHONY: clean
clean:
	rm -rf $(GOPATH)/bin/* $(GOPATH)/pkg/* bindata.go

.PHONY: cmd
cmd: $(SRC_FILES)
	@echo go run $^

$(GOPATH)/bin/octocatmd: $(SRC_FILES)
	@go install github.com/ixday/octocatmd

bindata.go: $(GOPATH)/bin/go-bindata $(DATA_DIR)/*
	@$< $(DATA_DIR)/

$(GOPATH)/src/github.com/jteeuwen/go-bindata:
	@go get -u github.com/jteeuwen/go-bindata


$(GOPATH)/bin/go-bindata: $(GOPATH)/src/github.com/jteeuwen/go-bindata
	@go install github.com/jteeuwen/go-bindata/go-bindata
