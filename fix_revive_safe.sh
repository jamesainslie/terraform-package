#!/bin/bash

# Safe script to fix revive issues without breaking syntax
echo "Fixing revive issues safely..."

# Pattern 1: Add comments to exported New functions (safe pattern matching)
sed -i '' '/^func NewDependenciesDataSource/i\
// NewDependenciesDataSource creates a new dependencies data source.
' internal/provider/dependencies_data_source.go

sed -i '' '/^func NewInstalledPackagesDataSource/i\
// NewInstalledPackagesDataSource creates a new installed packages data source.
' internal/provider/installed_packages_data_source.go

sed -i '' '/^func NewManagerInfoDataSource/i\
// NewManagerInfoDataSource creates a new manager info data source.
' internal/provider/manager_info_data_source.go

sed -i '' '/^func NewOutdatedPackagesDataSource/i\
// NewOutdatedPackagesDataSource creates a new outdated packages data source.
' internal/provider/outdated_packages_data_source.go

sed -i '' '/^func NewPackageInfoDataSource/i\
// NewPackageInfoDataSource creates a new package info data source.
' internal/provider/package_info_data_source.go

sed -i '' '/^func NewPackageResource/i\
// NewPackageResource creates a new package resource.
' internal/provider/package_resource.go

sed -i '' '/^func NewPackageSearchDataSource/i\
// NewPackageSearchDataSource creates a new package search data source.
' internal/provider/package_search_data_source.go

sed -i '' '/^func NewRegistryLookupDataSource/i\
// NewRegistryLookupDataSource creates a new registry lookup data source.
' internal/provider/registry_lookup_data_source.go

sed -i '' '/^func NewRepositoryPackagesDataSource/i\
// NewRepositoryPackagesDataSource creates a new repository packages data source.
' internal/provider/repository_packages_data_source.go

sed -i '' '/^func NewRepositoryResource/i\
// NewRepositoryResource creates a new repository resource.
' internal/provider/repository_resource.go

sed -i '' '/^func NewSecurityInfoDataSource/i\
// NewSecurityInfoDataSource creates a new security info data source.
' internal/provider/security_info_data_source.go

sed -i '' '/^func NewVersionHistoryDataSource/i\
// NewVersionHistoryDataSource creates a new version history data source.
' internal/provider/version_history_data_source.go

# Pattern 2: Fix unused ctx parameters (safe replacements)
find internal/provider internal/registry -name "*.go" -exec sed -i '' \
  -e 's/Metadata(ctx context\.Context,/Metadata(_ context.Context,/g' \
  -e 's/Schema(ctx context\.Context,/Schema(_ context.Context,/g' \
  -e 's/Configure(ctx context\.Context,/Configure(_ context.Context,/g' \
  -e 's/Resources(ctx context\.Context)/Resources(_ context.Context)/g' \
  -e 's/DataSources(ctx context\.Context)/DataSources(_ context.Context)/g' \
  -e 's/Functions(ctx context\.Context)/Functions(_ context.Context)/g' \
  {} \;

echo "Safe revive fixes applied."
