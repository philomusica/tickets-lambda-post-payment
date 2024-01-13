BINDIR:=./bin
BINARY:=bootstrap
ZIPFILE:=$(BINARY).zip
CMD:=./cmd
REPORT:=./report

$(BINARY):
	mkdir -p $(BINDIR)
	GOARCH=amd64 GOOS=linux go build -tags lambda.norpc -o $(BINDIR)/$(BINARY) $(CMD)

.PHONY: deps clean build deploy test vet fmt
deps:
	go get -u ./...

clean:
	rm -rf $(BINDIR)

deploy: $(BINARY)
ifeq ($(ARN),)
	@echo "Please set the ARN"
else
	(cd $(BINDIR) && zip -FS $(ZIPFILE) $(BINARY) ../*.ttf)
	aws lambda update-function-code --function-name $(ARN) --zip-file fileb://$(BINDIR)/$(ZIPFILE)
endif

test:
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
