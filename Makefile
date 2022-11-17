
build:
	go build ./cmd/cosmos-notifyer

run: build
	./cosmos-notifyer start

docker-deploy: docker-build docker-push

docker-build:
	docker build -t nysanetwork/cosmos-notifyer .

docker-push:
	docker push nysanetwork/cosmos-notifyer
