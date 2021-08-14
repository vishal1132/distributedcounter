.DEFAULT_GOAL=help
.PHONY: setup

image: ## build image. Don't know, redundant?
	@eval $(minikube -p minikube docker-env)
	@docker build -t distributed-likes .

run: ## run the distributed-likes cluster
	@eval $(minikube -p minikube docker-env)
	@docker build -t distributed-likes .
	@kubectl apply -f deployment/master_deployment.yml
	@kubectl apply -f deployment/master_service.yml

down: ## to take down the deployments and services
	@kubectl delete -f deployment/master_deployment.yml
	@kubectl delete -f deployment/master_service.yml

setup: ## first time running the code? use make setup
	@lefthook install

help:
	@echo "Usage:"
	@echo "  make [target...]"
	@echo ""
	@echo "Useful commands:"
	@grep -Eh '^[a-zA-Z._-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-30s %s\n", $$1, $$2}'
	@echo ""
