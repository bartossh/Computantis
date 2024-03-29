build-node:	
	cd src && CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/dedicated/node -ldflags="-s -w" cmd/node/main.go

build-webhooks:
	cd src && CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/dedicated/webhooks -ldflags="-s -w" cmd/webhooks/main.go

build-client:
	cd src && CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/dedicated/client -ldflags="-s -w" cmd/client/main.go

build-emulator:
	cd src && CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/dedicated/emulator -ldflags="-s -w" cmd/emulator/main.go

build-wallet:
	cd src && CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/dedicated/wallet -ldflags="-s -w" cmd/wallet/main.go

build-local: build-node build-webhooks build-client build-emulator build-wallet

build-all: build-local
	cd src && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/linux_x86/node -ldflags="-s -w" cmd/node/main.go
	cd src && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/linux_x86/webhooks -ldflags="-s -w" cmd/webhooks/main.go
	cd src && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/linux_x86/client -ldflags="-s -w" cmd/client/main.go
	cd src && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/linux_x86/emulator -ldflags="-s -w" cmd/emulator/main.go
	cd src && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/linux_x86/wallet -ldflags="-s -w" cmd/wallet/main.go

	cd src && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/linux_arm/node -ldflags="-s -w" cmd/node/main.go
	cd src && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/linux_arm/webhooks -ldflags="-s -w" cmd/webhooks/main.go
	cd src && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/linux_arm/client -ldflags="-s -w" cmd/client/main.go
	cd src && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/linux_arm/emulator -ldflags="-s -w" cmd/emulator/main.go
	cd src && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/linux_arm/wallet -ldflags="-s -w" cmd/wallet/main.go

	cd src && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/darwin_arm/node -ldflags="-s -w" cmd/node/main.go
	cd src && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/darwin_arm/webhooks -ldflags="-s -w" cmd/webhooks/main.go
	cd src && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/darwin_arm/client -ldflags="-s -w" cmd/client/main.go
	cd src && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/darwin_arm/emulator -ldflags="-s -w" cmd/emulator/main.go
	cd src && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/darwin_arm/wallet -ldflags="-s -w" cmd/wallet/main.go

	cd src && GOOS=linux GOARCH=arm GOARM=5 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/raspberry_pi_zero/node -ldflags="-s -w" cmd/node/main.go
	cd src && GOOS=linux GOARCH=arm GOARM=5 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/raspberry_pi_zero/webhooks -ldflags="-s -w" cmd/webhooks/main.go

	cd src && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/windows/wallet -ldflags="-s -w" cmd/wallet/main.go

build-tools:
	cd src && CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/dedicated/generator -ldflags="-s -w" -gcflags -m cmd/generator/main.go

build-tools-all: build-tools
	cd src && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/linux_x86/generator -ldflags="-s -w" cmd/generator/main.go

	cd src && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/linux_arm/generator -ldflags="-s -w" cmd/generator/main.go

	cd src && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/darwin_arm/generator -ldflags="-s -w" cmd/generator/main.go

documentation:
	./gendocs.sh

generate-secret:
	./secret.sh

generate-protobuf:
	protoc --proto_path=protobuf --go-grpc_out=src/protobufcompiled --go_out=src/protobufcompiled --go-grpc_opt=paths=source_relative --go_opt=paths=source_relative  computantistypes.proto wallet.proto gossip.proto notary.proto webhooks.proto addons.proto

generate-certificates:
	cd ./certificates && ./certificates.sh

run-node:
	./bin/dedicated/node -c conf/setup_example.yaml &

run-webhooks:
	./bin/dedicated/webhooks -c conf/setup_example.yaml &

run-client:
	./bin/dedicated/client -c conf/setup_example.yaml &

emulate-subscriber:
	./bin/dedicated/emulator -c conf/setup_example.yaml -d artefact/minmax.json subscriber &

emulate-publisher:
	./bin/dedicated/emulator -c conf/setup_example.yaml -d artefacts/data.json publisher &

run-emulate: emulate-subscriber emulate-publisher

run-all: run-node run-client run-webhooks emulate-subscriber emulate-publisher

start: build-local run-all

docker-dependencies:
	docker compose up -f docker_deployment/docker-compose.dependencies.yaml -d

# docker-up|down|logs all take into account the environment variable COMPOSE_PROFILES
# per https://docs.docker.com/compose/profiles/
# To run a complete demo with publisher and subscriber nodes, use:
#   COMPOSE_PROFILES=demo make docker-up|down|logs
# To run core services only, use:
#   make docker-up|down|logs
docker-up:
	docker compose -f docker_deployment/docker-compose.yaml up -d

docker-down:
	docker compose -f docker_deployment/docker-compose.yaml down

docker-logs:
	docker compose logs -f

docker-restart-node:
	docker-compose up -d --no-deps --build notary-node

docker-restart-webhooks:
	docker-compose up -d --no-deps --build webhooks-node

docker-build-all: docker-build-node docker-build-webhooks docker-build-client docker-build-subscriber docker-build-publisher

docker-build-node:
	docker compose build notary-node

docker-build-webhooks:
	docker compose build webhooks-node

docker-build-client:
	docker compose build client-node

docker-build-subscriber:
	docker compose build subscriber-node

docker-build-publisher:
	docker compose build publisher-node

scan:
	govulncheck ./src/...

k8s-set-konfig:
	export KUBECONFIG=${PWD}/kubeconfig.yaml

k8s-set-secret-wallet: k8s-set-konfig
	kubectl create secret generic wallet-secret --from-file=walet=aretfacts/test_wallet

k8s-docker-auth: k8s-set-konfig
	kubectl create secret generic regcred --from-file=.dockerconfigjson=${HOME}/.docker/config.json --type=kubernetes.io/dockerconfigjson
