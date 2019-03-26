
ENVIRONMENT  ?= prod
AWS_REGION   ?= eu-west-1
AWS_PROFILE  ?=
PROJECT      ?= itops
OWNER        ?= cloudops
SERVICE_NAME ?= ec2scheduler
S3_BUCKET    ?=

###

deploy: build deploy

build:
	cd source/scheduler-disable; go test -v -cover && GOOS=linux go build -o main && zip handler.zip main
	cd source/scheduler-set; GOOS=linux go build -o main && zip handler.zip main
	cd source/scheduler-status; GOOS=linux go build -o main && zip handler.zip main
	cd source/scheduler-suspend; GOOS=linux go build -o main && zip handler.zip main
	cd source/scheduler-unsuspend; GOOS=linux go build -o main && zip handler.zip main
	cd source/scheduler-suspend-mon; GOOS=linux go build -o main && zip handler.zip main
	cd source/scheduler; go test -v -cover && GOOS=linux go build -o main && zip handler.zip main
	mkdir -p dist
	aws cloudformation package \
		--template-file sam.yaml \
		--output-template-file dist/sam.yaml \
		--s3-bucket $(S3_BUCKET) \
		--s3-prefix $(PROJECT)/$(SERVICE_NAME)

deploy:
	aws cloudformation deploy \
		--template-file dist/sam.yaml \
		--stack-name ec2scheduler \
		--tags \
			Environment=$(ENVIRONMENT) \
			Project=$(PROJECT) \
			Owner=$(OWNER) \
		--capabilities CAPABILITY_IAM
	rm -rf dist source/*/main source/*/handler.zip

