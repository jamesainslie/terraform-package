provider "package" {
  # Basic configuration
  default_manager = "auto"      # auto-detects based on OS (darwin=brew, linux=apt, windows=winget)
  assume_yes      = true        # run non-interactively
  sudo_enabled    = true        # enable sudo on Unix systems
  update_cache    = "on_change" # update cache when packages change

  # Optional: Custom binary paths
  # brew_path    = "/opt/homebrew/bin/brew"
  # apt_get_path = "/usr/bin/apt-get"
  # winget_path  = "C:\\Windows\\System32\\winget.exe"

  # Optional: Timeout configuration
  # lock_timeout = "10m"
}
