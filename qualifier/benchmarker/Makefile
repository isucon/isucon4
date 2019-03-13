GIT_COMMIT=$(shell git log --oneline -n 1 . | awk '{print $$1}')
DUTMH=$(shell ./md5 sql/dummy_users.tsv)
DUUTMH=$(shell ./md5 sql/dummy_users_used.tsv)

debugmode = false
LD_FLAGS = \
	-X main.GIT_COMMIT=${GIT_COMMIT} \
	-X github.com/KentaKudo/isucon4/qualifier/benchmarker/user.DummyUsersTSVMD5=${DUTMH} \
	-X github.com/KentaKudo/isucon4/qualifier/benchmarker/user.DummyUsersUsedTSVMD5=${DUUTMH} \
	-X github.com/KentaKudo/isucon4/qualifier/benchmarker/user.DebugMode=${debugmode} \
	-X main.DebugMode=${debugmode} \
	-X main.SkipMetadataMode=${skipmetadata}

ifeq ($(filter true false,${debugmode}),)
  $(error debugmode should be true or false)
endif


test:
	@go test -v ./...

benchmarker: deps user/dummy_users.go
	@go build -ldflags "${LD_FLAGS}"

deps:
	@go mod download

debug: debugmode=true
debug: benchmarker
	@tput setaf 3
	@echo -n '! '
	@tput sgr0
	@echo "You have enabled DEBUG mode."

release: debugmode=false
release: benchmarker
	@tput setaf 2
	@echo -n '* '
	@tput sgr0
	@echo "Built for release."

release-nometadata: skipmetadata=true
release-nometadata: debugmode=false
release-nometadata: benchmarker
	@tput setaf 2
	@echo -n '* '
	@tput sgr0
	@echo "Built for release, without metadata check."

.PHONEY: test release deps
