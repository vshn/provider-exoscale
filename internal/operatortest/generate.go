//go:build generate

package operatortest

// Generate mock for egoscale
//go:generate go run github.com/vektra/mockery/v2 --srcpkg=github.com/exoscale/egoscale/v2/oapi --name=ClientWithResponsesInterface --outpkg=operatortest --output . --filename egoscale.go
