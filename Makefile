
ENVIRONMENT  ?= prod
AWS_REGION   ?= eu-west-1
AWS_PROFILE  ?=
PROJECT      ?= itops
OWNER        ?= cloudops
SERVICE_NAME ?= ec2scheduler
S3_BUCKET    ?=
FUNCTIONS    = scheduler scheduler-disable scheduler-set scheduler-status scheduler-suspend scheduler-unsuspend scheduler-suspend-mon

###

deploy: build package deploy

build:
	@docker run --rm \
		-v $(PWD)/source:/src \
		-w /src \
		-e FUNCTIONS="${FUNCTIONS}" \
		golang:stretch sh -c \
			'apt-get update && apt-get install -y zip && \
			for f in ${FUNCTIONS}; do \
				echo "\n▸ $$f - building code..." && \
				cd /src/$$f && go test -v -cover && go build -ldflags="-s -w" -o main && \
				zip handler.zip main && \
				rm main && cd ../.. && \
				echo "▸ $$f - build done..."; \
			done'

package:
	mkdir -p build
	aws cloudformation package \
		--template-file template.yaml \
		--output-template-file build/template.yaml \
		--s3-bucket $(S3_BUCKET) \
		--s3-prefix $(PROJECT)/$(SERVICE_NAME)

deploy:
	aws cloudformation deploy \
		--template-file build/template.yaml \
		--stack-name ec2scheduler \
		--tags \
			Environment=$(ENVIRONMENT) \
			Project=$(PROJECT) \
			Owner=$(OWNER) \
		--capabilities CAPABILITY_IAM
	rm -rf build source/*/main source/*/handler.zip

clean:
	rm -rf build source/*/main source/*/handler.zip

# eof
