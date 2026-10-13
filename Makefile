WAILS := $(shell which wails 2>/dev/null || echo $(HOME)/go/bin/wails)
BIN_DIR := ./bin
BINARY := go-countdown

# Linux needs the app compiled against the system's webkit2gtk headers.
# Ubuntu 24.04+ only ships the 4.1 dev package (4.0 was dropped), so builds
# targeting it need this tag — see `wails doctor`, which still reports
# "libwebkit ... Not Found" on such systems even though the build works,
# since that check only recognizes the older 4.0 package name.
LINUX_TAGS := webkit2_41

# The nearest git tag (e.g. v1.2.0), with -N-gHASH appended if HEAD is N
# commits past it, and -dirty if the worktree has uncommitted changes.
# Falls back to "dev" outside a git checkout or before any tag exists, so a
# fresh clone still builds instead of erroring out.
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/ashokhin/go-countdown/internal/app.Version=$(VERSION)

.PHONY: build build-linux build-windows build-all run clean test vet

build: build-all

# Wails always writes to build/bin/ itself (only the filename is
# configurable, not the directory), so this just moves the result into
# ./bin afterwards.
# -nopackage is deliberately NOT used here: on Windows that step is also
# what compiles build/windows/icon.ico into a .syso resource embedded in the
# .exe (Explorer/taskbar icon) — it doesn't require or invoke NSIS unless
# -nsis is also passed, so this stays a single portable .exe either way.
build-linux:
	mkdir -p $(BIN_DIR)
	$(WAILS) build -tags $(LINUX_TAGS) -ldflags "$(LDFLAGS)" -o $(BINARY)
	mv build/bin/$(BINARY) $(BIN_DIR)/$(BINARY)

# Cross-compiles from Linux without needing a Windows machine or a CGO
# cross-compiler: Wails' Windows backend talks to WebView2 through plain Go
# syscalls, not CGO, so a normal GOOS=windows build is enough.
build-windows:
	mkdir -p $(BIN_DIR)
	$(WAILS) build -platform windows/amd64 -ldflags "$(LDFLAGS)" -o $(BINARY).exe
	mv build/bin/$(BINARY).exe $(BIN_DIR)/$(BINARY).exe

build-all: build-linux build-windows

run: build-linux
	$(BIN_DIR)/$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR) build/bin
