# go 相关的配置文件和命令

GO := go
# 默认最小覆盖率60%
ifeq ($(origin COVERAGE),undefined)
COVERAGE := 60
endif

# 自动添加/移除依赖包
.PHONY: go.tidy
go.tidy:
	$(GO) mod tidy


# 代码格式化
.PHONY: go.format
go.format:
	@echo "===========> Formating codes"
	$(FIND) -type f -name '*.go' | $(XARGS) gofmt -s -w
	$(FIND) -type f -name '*.go' | $(XARGS) goimports -w -local $(ROOT_PACKAGE)
	$(FIND) -type f -name '*.go' | $(XARGS) golines -w --max-len=120 --no-reformat-tags --shorten-comments --ignore-generated .
	$(GO) mod edit -fmt
	$(GIT) add -A .

