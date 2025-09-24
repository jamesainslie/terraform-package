// MIT License
//
// Copyright (c) 2025 Terraform Package Provider Contributors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package services

import (
	"runtime"

	"github.com/jamesainslie/terraform-package/internal/executor"
)

// NewServiceDetector creates a platform-appropriate service detector
func NewServiceDetector(executor executor.Executor) ServiceDetector {
	mapping := GetDefaultMapping()
	healthChecker := NewDefaultHealthChecker(executor)

	manager := newPlatformServiceDetector(executor, mapping, healthChecker)
	return manager
}

// NewServiceDetectorWithMapping creates a service detector with custom mapping
func NewServiceDetectorWithMapping(executor executor.Executor, mapping *PackageServiceMapping) ServiceDetector {
	healthChecker := NewDefaultHealthChecker(executor)

	manager := newPlatformServiceDetector(executor, mapping, healthChecker)
	return manager
}

// NewServiceManager creates a platform-appropriate service manager
func NewServiceManager(executor executor.Executor) ServiceManager {
	mapping := GetDefaultMapping()
	healthChecker := NewDefaultHealthChecker(executor)

	return newPlatformServiceDetector(executor, mapping, healthChecker)
}

// NewServiceManagerWithMapping creates a service manager with custom mapping
func NewServiceManagerWithMapping(executor executor.Executor, mapping *PackageServiceMapping) ServiceManager {
	healthChecker := NewDefaultHealthChecker(executor)

	return newPlatformServiceDetector(executor, mapping, healthChecker)
}

// GetCurrentPlatform returns the current platform
func GetCurrentPlatform() Platform {
	switch runtime.GOOS {
	case "darwin":
		return PlatformDarwin
	case "linux":
		return PlatformLinux
	case "windows":
		return PlatformWindows
	default:
		return PlatformLinux // fallback
	}
}
