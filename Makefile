.DEFAULT_ALL = all

include scripts/common.mk
include scripts/golang.mk
include scripts/docker.mk



.PHONY: all
all: go.tidy go.format go.lint

