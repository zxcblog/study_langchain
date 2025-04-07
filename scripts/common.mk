# 通用的配置文件，用来配置所有makefile文件都会使用到的配置信息
SHELL := /bin/bash
GIT := git
GOOS = $(shell go env GOOS)

# 使用 MAKEFILE_LIST 来获取 common.md 文件的位置
# 如果是在windows系统时，ROOT_DIR可能会获取异常，需要特殊指定
COMMON_SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
ROOT_DIR := $(abspath $(shell cd $(COMMON_SELF_DIR)/../.. && pwd -P))
ifeq ($(GOOS),windows)
	ROOT_DIR = E:/golang/bg-ai
	ROOT_DIR = E:/code/study_langchain
	GO_OUT_EXT := .exe
endif

# linux 命令
FIND := find . ! -path './docker/*' ! -path './scripts/*'
XARGS := xargs -r

