package config

import (
	"fmt"
	"os"
)

// GenerateCompletion generates shell completion scripts
func GenerateCompletion(shellType string) error {
	switch shellType {
	case "bash":
		return generateBashCompletion()
	case "zsh":
		return generateZshCompletion()
	case "fish":
		return generateFishCompletion()
	default:
		return fmt.Errorf("unsupported shell type: %s", shellType)
	}
}

func generateBashCompletion() error {
	bashCompletion := `
# Bash completion for hubsync

_hubsync_completions() {
  local cur prev opts
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  
  # List of options with values
  local opts_with_values="--username --password --repository --namespace --content --maxContent --outputPath --concurrency --timeout --retryCount --retryDelay --logLevel --logFile --completion"
  
  # List of all options
  local opts="--help --version ${opts_with_values}"
  
  # Handle special completion cases
  case "${prev}" in
    --logLevel)
      COMPREPLY=( $(compgen -W "debug info warn error" -- ${cur}) )
      return 0
      ;;
    --completion)
      COMPREPLY=( $(compgen -W "bash zsh fish" -- ${cur}) )
      return 0
      ;;
    *)
      if [[ ${cur} == -* ]]; then
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
      fi
      ;;
  esac
  
  return 0
}

complete -F _hubsync_completions hubsync
`
	// Check if output directory is specified
	outputFile := os.Getenv("COMPLETION_OUTPUT")
	if outputFile != "" {
		return os.WriteFile(outputFile, []byte(bashCompletion), 0o644)
	}

	// If no output specified, try to install to common bash completion directories
	completionDirs := []string{
		"/etc/bash_completion.d/",
		"/usr/local/etc/bash_completion.d/",
		"/usr/share/bash-completion/completions/",
		os.Getenv("HOME") + "/.local/share/bash-completion/completions/",
	}

	// Try to install to the first writable directory
	for _, dir := range completionDirs {
		if _, err := os.Stat(dir); err == nil {
			// Directory exists, try to write the file
			path := dir + "hubsync"
			err := os.WriteFile(path, []byte(bashCompletion), 0o644)
			if err == nil {
				fmt.Printf("Bash completion script installed to %s\n", path)
				return nil
			}
		}
	}

	// If we couldn't install to any directory, just print the script
	fmt.Print(bashCompletion)
	fmt.Println("\n# Bash completion script couldn't be installed automatically.")
	fmt.Println("# Add this to your ~/.bashrc or copy to /etc/bash_completion.d/")
	return nil
}

func generateZshCompletion() error {
	zshCompletion := `
#compdef hubsync

_arguments \
  '--username[Docker registry username]:username:' \
  '--password[Docker registry password]:password:' \
  '--repository[Target repository address]:repository:' \
  '--namespace[Target namespace]:namespace:' \
  '--content[JSON content with images to sync]:content:_files' \
  '--maxContent[Maximum number of images to process]:maxContent:' \
  '--outputPath[Output file path]:outputPath:_files' \
  '--concurrency[Maximum concurrent operations]:concurrency:' \
  '--timeout[Operation timeout]:timeout:' \
  '--retryCount[Number of retries for failed operations]:retryCount:' \
  '--retryDelay[Delay between retries]:retryDelay:' \
  '--logLevel[Log level]:level:(debug info warn error)' \
  '--logFile[Log to file in addition to stdout]:logFile:_files' \
  '--completion[Generate shell completion script]:shell:(bash zsh fish)' \
  '--help[Show help message]' \
  '--version[Show version information]'
`
	// Check if output directory is specified
	outputFile := os.Getenv("COMPLETION_OUTPUT")
	if outputFile != "" {
		return os.WriteFile(outputFile, []byte(zshCompletion), 0o644)
	}

	// If no output specified, try to install to common zsh completion directories
	completionDirs := []string{
		"/usr/local/share/zsh/site-functions/",
		"/usr/share/zsh/site-functions/",
		"/usr/share/zsh/vendor-completions/",
		os.Getenv("HOME") + "/.zsh/completions/",
	}

	// Try to install to the first writable directory
	for _, dir := range completionDirs {
		if _, err := os.Stat(dir); err == nil {
			// Directory exists, try to write the file
			path := dir + "_hubsync"
			err := os.WriteFile(path, []byte(zshCompletion), 0o644)
			if err == nil {
				fmt.Printf("Zsh completion script installed to %s\n", path)
				return nil
			}
		}
	}

	// If we couldn't install to any directory, just print the script
	fmt.Print(zshCompletion)
	fmt.Println("\n# Zsh completion script couldn't be installed automatically.")
	fmt.Println("# Copy this script to a directory in your fpath, e.g. ~/.zsh/completions/_hubsync")
	fmt.Println("# Make sure to add 'fpath=(~/.zsh/completions $fpath)' to your .zshrc")
	return nil
}

func generateFishCompletion() error {
	fishCompletion := `
# Fish shell completion for hubsync

complete -c hubsync -l username -d "Docker registry username"
complete -c hubsync -l password -d "Docker registry password"
complete -c hubsync -l repository -d "Target repository address"
complete -c hubsync -l namespace -d "Target namespace" 
complete -c hubsync -l content -d "JSON content with images to sync" -r
complete -c hubsync -l maxContent -d "Maximum number of images to process"
complete -c hubsync -l outputPath -d "Output file path" -r
complete -c hubsync -l concurrency -d "Maximum concurrent operations"
complete -c hubsync -l timeout -d "Operation timeout"
complete -c hubsync -l retryCount -d "Number of retries for failed operations"
complete -c hubsync -l retryDelay -d "Delay between retries"
complete -c hubsync -l logLevel -d "Log level" -r -a "debug info warn error"
complete -c hubsync -l logFile -d "Log to file in addition to stdout" -r
complete -c hubsync -l completion -d "Generate shell completion script" -r -a "bash zsh fish"
complete -c hubsync -l help -s h -d "Show help message"
complete -c hubsync -l version -s v -d "Show version information"
`
	// Check if output directory is specified
	outputFile := os.Getenv("COMPLETION_OUTPUT")
	if outputFile != "" {
		return os.WriteFile(outputFile, []byte(fishCompletion), 0o644)
	}

	// If no output specified, try to install to common fish completion directories
	completionDirs := []string{
		"/usr/local/share/fish/vendor_completions.d/",
		"/usr/share/fish/vendor_completions.d/",
		os.Getenv("HOME") + "/.config/fish/completions/",
	}

	// Try to install to the first writable directory
	for _, dir := range completionDirs {
		if _, err := os.Stat(dir); err == nil {
			// Directory exists, try to write the file
			path := dir + "hubsync.fish"
			err := os.WriteFile(path, []byte(fishCompletion), 0o644)
			if err == nil {
				fmt.Printf("Fish completion script installed to %s\n", path)
				return nil
			}
		}
	}

	// If we couldn't install to any directory, just print the script
	fmt.Print(fishCompletion)
	fmt.Println("\n# Fish completion script couldn't be installed automatically.")
	fmt.Println("# Copy this script to ~/.config/fish/completions/hubsync.fish")
	return nil
}
