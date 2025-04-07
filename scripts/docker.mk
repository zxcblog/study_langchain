# docker 镜像命令相关

DOCKER := docker

# 停止运行所有容器
.PHONY: docker.all.stop
docker.all.stop:
	$(DOCKER) stop $(shell $(DOCKER) ps -q)


# 删除所有容器
.PHONY: docker.all.del
docker.all.del:
	$(DOCKER) rm $(shell $(DOCKER) ps -aq)


# 根据docker文件夹下的yaml文件启动
# 命令会将%为文件名称 例： make docker.mysql
.PHONY: docker.%
docker.%:
	$(DOCKER) compose -f $(ROOT_DIR)/docker/$(*).yaml up -d


# 停止容器
.PHONY: docker.%.stop
docker.%.stop:
	$(DOCKER) compose -f $(ROOT_DIR)/docker/$(*).yaml stop


# 停止并删除容器
.PHONY: docker.%.down
docker.%.down:
	$(DOCKER) compose -f $(ROOT_DIR)/docker/$(*).yaml down


# 删除容器对应的持久化目录
.PHONY: docker.%.delfile
docker.%.delfile:
	@$(MAKE) docker.$(*).down
	@$(shell rm -rf $(ROOT_DIR)/docker/data/$*)


# 重启容器
# 使用restart进行重启时，不会使用新的yaml的新的改动
.PHONY: docker.%.restart
docker.%.restart:
	@$(MAKE) docker.$(*).down
	@$(MAKE) docker.$(*)

# 启动ollama特殊对待
.PHONY: docker.ollama
docker.ollama:
	$(DOCKER) compose -f $(ROOT_DIR)/docker/ollama.yaml up -d
	docker exec -it ollama ollama create qwen2.5:3b -f /usr/local/ollama/Modelfile
	docker exec -it ollama ollama run qwen2.5:3b