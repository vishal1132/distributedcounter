image:
	@eval $(minikube -p minikube docker-env)
	@docker build -t distributed-likes .

run:
	@eval $(minikube -p minikube docker-env)
	@docker build -t distributed-likes .
	@kubectl apply -f master_deployment.yml
	@kubectl apply -f master_service.yml

down:
	@kubectl delete -f master_deployment.yml
	@kubectl delete -f master_service.yml
# eval $(minikube -p minikube docker-env)