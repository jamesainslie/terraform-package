terraform {
  required_providers {
    package = {
      source = "jamesainslie/package"
    }
  }
}

provider "pkg" {
  default_manager = "auto"
  assume_yes      = true
}

# Example: Install development tools
resource "pkg_package" "git" {
  name    = "git"
  state   = "present"
  version = ""    # latest version
  pin     = false # allow updates

  timeouts {
    create = "10m"
    delete = "5m"
  }
}

# Example: Install with version pinning
resource "pkg_package" "node" {
  name    = "nodejs"
  state   = "present"
  version = "20.*" # any 20.x version
  pin     = true   # prevent updates

  # Platform-specific package names
  aliases = {
    darwin  = "node@20"
    linux   = "nodejs"
    windows = "OpenJS.NodeJS.LTS"
  }
}

# Example: Custom repository (Homebrew tap)
resource "pkg_repo" "custom_tap" {
  manager = "brew"
  name    = "homebrew/cask-fonts"
  uri     = "homebrew/cask-fonts"
}

# Example: Install package from custom repository
resource "pkg_package" "custom_font" {
  name  = "font-fira-code"
  state = "present"

  depends_on = [pkg_repo.custom_tap]
}