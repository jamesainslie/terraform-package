#!/bin/bash

# Script to automatically fix common revive issues
echo "Fixing revive issues..."

# Pattern 1: Add comments to exported functions starting with "New"
find internal/provider -name "*.go" -exec sed -i '' \
  -e 's/^func New\([A-Za-z]*\)DataSource() datasource\.DataSource {$/\/\/ New\1DataSource creates a new \1 data source.\
func New\1DataSource() datasource.DataSource {/g' \
  -e 's/^func New\([A-Za-z]*\)Resource() resource\.Resource {$/\/\/ New\1Resource creates a new \1 resource.\
func New\1Resource() resource.Resource {/g' \
  {} \;

# Pattern 2: Add comments to exported Provider functions
find internal/provider -name "*.go" -exec sed -i '' \
  -e 's/^func New(version string) func() provider\.Provider {$/\/\/ New creates a new package provider instance.\
func New(version string) func() provider.Provider {/g' \
  {} \;

# Pattern 3: Fix unused parameters by renaming ctx to _
find internal/provider internal/registry -name "*.go" -exec sed -i '' \
  -e 's/func ([^)]*) \([A-Za-z]*\)(ctx context\.Context,/func (\&) \1(_ context.Context,/g' \
  -e 's/func ([^)]*) \([A-Za-z]*\)(_ context\.Context,/func (\&) \1(_ context.Context,/g' \
  {} \;

# Pattern 4: Add comments to exported methods (CRUD operations)
find internal/provider -name "*.go" -exec sed -i '' \
  -e 's/^func (r \*\([A-Za-z]*\)Resource) Create(/\/\/ Create creates a new \1 resource.\
func (r *\1Resource) Create(/g' \
  -e 's/^func (r \*\([A-Za-z]*\)Resource) Update(/\/\/ Update updates an existing \1 resource.\
func (r *\1Resource) Update(/g' \
  -e 's/^func (r \*\([A-Za-z]*\)Resource) Delete(/\/\/ Delete removes a \1 resource.\
func (r *\1Resource) Delete(/g' \
  -e 's/^func (r \*\([A-Za-z]*\)Resource) ImportState(/\/\/ ImportState imports an existing \1 resource.\
func (r *\1Resource) ImportState(/g' \
  {} \;

# Pattern 5: Add comments to exported data source methods
find internal/provider -name "*.go" -exec sed -i '' \
  -e 's/^func (d \*\([A-Za-z]*\)DataSource) Metadata(/\/\/ Metadata returns the data source metadata.\
func (d *\1DataSource) Metadata(/g' \
  -e 's/^func (d \*\([A-Za-z]*\)DataSource) Schema(/\/\/ Schema returns the data source schema.\
func (d *\1DataSource) Schema(/g' \
  -e 's/^func (d \*\([A-Za-z]*\)DataSource) Configure(/\/\/ Configure configures the data source with provider data.\
func (d *\1DataSource) Configure(/g' \
  {} \;

echo "Revive fixes applied. Run golangci-lint to check results."
