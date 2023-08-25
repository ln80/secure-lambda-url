

rotation/validate:
	sam validate --lint --template-file stack/rotation/template.yaml

rotation/build:
	sam build --cached \
		--template-file stack/rotation/template.yaml \
		--base-dir . \
		--build-dir .aws-sam/rotation

rotation/package: 
	sam package \
		--template .aws-sam/rotation/template.yaml \
		--output-template-file stack/rotation/packaged.yaml \
		--s3-bucket ln80-sam-pkgs

rotation/publish:
	sam publish \
		--template stack/rotation/packaged.yaml \
		--region eu-west-1

extension/build:
	sam build --cached \
		--template-file stack/extension/template.yaml \
		--base-dir . \
		--build-dir .aws-sam/extension

extension/package: 
	sam package \
		--template .aws-sam/extension/template.yaml \
		--output-template-file stack/extension/packaged.yaml \
		--s3-bucket ln80-sam-pkgs

extension/publish:
	sam publish \
		--template stack/extension/packaged.yaml \
		--region eu-west-1

# internally used by extension/build 
build-LambdaExtensionLayer:
	cd ./stack/extension ;\
	GOOS=linux GOARCH=amd64 go build -o $(ARTIFACTS_DIR)/extensions/secure-lambda-url-extension
	chmod +x $(ARTIFACTS_DIR)/extensions/secure-lambda-url-extension

# internally used by extension/build 
build-LambdaExtensionArm64Layer:
	cd ./stack/extension ;\
	GOOS=linux GOARCH=arm64 go build -o $(ARTIFACTS_DIR)/extensions/secure-lambda-url-extension-arm64
	chmod +x $(ARTIFACTS_DIR)/extensions/secure-lambda-url-extension-arm64


# get-layer:
# 	URL="`aws lambda get-layer-version --layer-name secure-lambda-url-extension --version-number 5 --query Content.Location --output text`"; \
# 	curl $$URL -o layer.zip


example/synth:
	@ cd examples/cdk; \
	cdk synthesize

example/deploy:
	@ cd examples/cdk; \
	cdk deploy --all

example/destroy:
	@ cd examples/cdk; \
	cdk destroy ExampleStack
