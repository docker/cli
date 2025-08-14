package service

import si "github.com/docker/cli/cli/command/internal/service"

var (
	NewServiceCommand             = si.NewServiceCommand
	ValidateSingleGenericResource = si.ValidateSingleGenericResource
	ParseGenericResources         = si.ParseGenericResources
	AppendServiceStatus           = si.AppendServiceStatus
	WaitOnService                 = si.WaitOnService
	ParseSecrets                  = si.ParseSecrets
	ParseConfigs                  = si.ParseConfigs
)
