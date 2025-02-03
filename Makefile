deps:
	@go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo
	@go install -mod=mod github.com/onsi/gomega
	@go mod tidy

e2e-install-rancher: deps
	ginkgo --label-filter install -r -v ./test/e2e

# Qase commands
create-qase-run: deps
	@go run tests/helper/qase/helper_qase.go -create
delete-qase-run: deps
	@go run tests/helper/qase/helper_qase.go -delete
publish-qase-run: deps
	@go run tests/helper/qase/helper_qase.go -publish
