# Codewisdom Train-Ticket system

Repo=codewisdom
Tag=latest
Namespace="default"
DeployArgs=""


# build image
.PHONY: build
build: clean-image package build-image

.PHONY: package package-java package-go package-python test compatibility-test
package: package-java package-go package-python

package-java:
	@mvn clean package -Dmaven.test.skip=true

package-go:
	@mkdir -p ts-route-plan-service/target ts-station-service/target
	@cd ts-route-plan-service && go test -mod=readonly ./... && go build -mod=readonly -o target/ts-route-plan-service ./cmd/server
	@cd ts-station-service && go test -mod=readonly ./... && go build -mod=readonly -o target/ts-station-service ./cmd/server

package-python:
	@cd ts-travel-plan-service && uv sync --frozen && uv run --frozen pytest && uv run --frozen python -m compileall -q app
	@cd ts-consign-service && uv sync --frozen && uv run --frozen pytest && uv run --frozen python -m compileall -q app

compatibility-test:
	@python3 benchmark-ground-truth/verify.py
	@python3 benchmark-ground-truth/generate_graph.py --check
	@cd ts-travel-plan-service && uv run --frozen pytest ../compatibility-tests

test: package-go package-python compatibility-test

.PHONY: build-image
build-image:
	@hack/build-image.sh $(Repo) $(Tag)

# push image
.PHONY: push-image
push-image:
	@hack/push-image.sh $(Repo)

.PHONY: publish-image
publish-image:
	@script/publish-docker-images.sh $(Repo) $(Tag)

# deploy
# DeployArgs ""                    : deploy train-ticket with all-in-one mysql cluster
# DeployArgs "--independent-db"    : deploy train-ticket with mysql cluster each service
# DeployArgs "--with-monitoring"   : deploy train-ticket with prometheus
# DeployArgs "--with-tracing"      : deploy train-ticket with skywalking
# DeployArgs "--all"               : deploy train-ticket with mysql cluster each service
.PHONY: deploy
deploy:
	@hack/deploy/deploy.sh $(Namespace) "$(DeployArgs)"

# deploy
.PHONY: reset-deploy
reset-deploy:
	@hack/deploy/reset.sh $(Namespace)

.PHONY: clean
clean:
	@mvn clean
	@cd ts-route-plan-service && go clean
	@cd ts-station-service && go clean
	@hack/clean-image.sh $(Repo)

# clean image
.PHONY: clean-image
clean-image:
	@hack/clean-image.sh $(Repo)
