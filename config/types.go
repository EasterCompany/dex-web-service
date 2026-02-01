package config

import (
	sharedConfig "github.com/EasterCompany/dex-go-utils/config"
)

// Aliases to shared types in dex-go-utils for backward compatibility
type ServiceMapConfig = sharedConfig.ServiceMapConfig
type ServiceType = sharedConfig.ServiceType
type ServiceEntry = sharedConfig.ServiceEntry
type OptionsConfig = sharedConfig.OptionsConfig
type DiscordOptions = sharedConfig.DiscordOptions
type SystemConfig = sharedConfig.SystemConfig
type CPUInfo = sharedConfig.CPUInfo
type GPUInfo = sharedConfig.GPUInfo
type StorageInfo = sharedConfig.StorageInfo
type PackageInfo = sharedConfig.PackageInfo
