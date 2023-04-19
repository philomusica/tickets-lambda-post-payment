BINDIR:=./bin
ZIPFILE:=function.zip
BINARY:=main
CMD:=./cmd
REPORT:=./report

$(BINARY):
	mkdir -p $(BINDIR) 
#	GOOS=linux GOARCH=amd64 go build -gcflags="-m" -o $(BINDIR)/$(BINARY) $(CMD)
	GOOS=linux GOARCH=amd64 go build -o $(BINDIR)/$(BINARY) $(CMD)

.PHONY: deps clean deploy test vet fmt
deps:
	go get -u ./...

clean:
	rm -rf $(BINDIR)

deploy: $(BINARY)
ifeq ($(ARN),)
	@echo "Please set the ARN"
else
	zip -r $(ZIPFILE) $(BINDIR) ./*.ttf
	aws lambda update-function-code --function-name $(ARN) --zip-file fileb://$(ZIPFILE)
endif

test:
	#go test -v -cover ./...
	go test -cover ./...

cover:
	mkdir -p $(REPORT)
	go test ./... -coverprofile $(REPORT)/cover.out
	go tool cover -html=$(REPORT)/cover.out -o $(REPORT)/index.html
	cd $(REPORT) && python3 -m http.server 8000
	

vet:
	go vet ./...

fmt:
	go fmt ./...
