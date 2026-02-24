package cmdutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/ir"
)

// DetectComposeCommand returns the docker compose command to use.
// It tries "docker compose" (v2) first, then falls back to "docker-compose" (v1).
func DetectComposeCommand() ([]string, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker not found in PATH. Install Docker to deploy")
	}

	composeCmd := []string{"docker", "compose"}
	if err := exec.Command("docker", "compose", "version").Run(); err != nil {
		if _, err := exec.LookPath("docker-compose"); err != nil {
			return nil, fmt.Errorf("neither 'docker compose' nor 'docker-compose' found. Install Docker Compose")
		}
		composeCmd = []string{"docker-compose"}
	}
	return composeCmd, nil
}

// DeployDocker builds and starts containers using docker compose.
func DeployDocker(app *ir.Application, outputDir string, dryRun bool) error {
	composePath := filepath.Join(outputDir, "docker-compose.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose.yml not found. Run 'human build <file>' first")
	}

	composeCmd, err := DetectComposeCommand()
	if err != nil {
		return err
	}

	// Check .env file
	envPath := filepath.Join(outputDir, ".env")
	envExamplePath := filepath.Join(outputDir, ".env.example")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if _, err := os.Stat(envExamplePath); err == nil {
			fmt.Println(cli.Warn("No .env file found. Copying .env.example → .env"))
			fmt.Println(cli.Warn("Review and update .env with production values before deploying to production."))
			if !dryRun {
				content, readErr := os.ReadFile(envExamplePath)
				if readErr != nil {
					return fmt.Errorf("reading .env.example: %w", readErr)
				}
				if writeErr := os.WriteFile(envPath, content, 0644); writeErr != nil {
					return fmt.Errorf("creating .env: %w", writeErr)
				}
			}
		}
	}

	// Build step
	buildArgs := append(composeCmd, "build")
	fmt.Println(cli.Info(fmt.Sprintf("Step 1/2: %s", strings.Join(buildArgs, " "))))
	if dryRun {
		fmt.Println(cli.Info("  (dry-run — skipped)"))
	} else {
		if err := RunCommand(outputDir, buildArgs[0], buildArgs[1:]...); err != nil {
			return fmt.Errorf("Docker build failed: %w", err)
		}
	}

	// Up step
	upArgs := append(composeCmd, "up", "-d")
	fmt.Println(cli.Info(fmt.Sprintf("Step 2/2: %s", strings.Join(upArgs, " "))))
	if dryRun {
		fmt.Println(cli.Info("  (dry-run — skipped)"))
	} else {
		if err := RunCommand(outputDir, upArgs[0], upArgs[1:]...); err != nil {
			return fmt.Errorf("Docker deploy failed: %w", err)
		}
	}

	if dryRun {
		fmt.Println(cli.Success("Dry run complete — no changes were made."))
	} else {
		fmt.Println(cli.Success(fmt.Sprintf("Deployed %s via Docker.", app.Name)))
		fmt.Println(cli.Info("  Run 'docker compose ps' in .human/output/ to check status."))
		fmt.Println(cli.Info("  Run 'docker compose logs -f' to view logs."))
		fmt.Println(cli.Info("  Run 'docker compose down' to stop."))
	}
	return nil
}

// StopDocker stops docker compose containers in the output directory.
func StopDocker(outputDir string) error {
	composeCmd, err := DetectComposeCommand()
	if err != nil {
		return err
	}
	downArgs := append(composeCmd, "down")
	return RunCommand(outputDir, downArgs[0], downArgs[1:]...)
}

// DockerStatus shows docker compose status in the output directory.
func DockerStatus(outputDir string) error {
	composeCmd, err := DetectComposeCommand()
	if err != nil {
		return err
	}
	psArgs := append(composeCmd, "ps")
	return RunCommandSilent(outputDir, psArgs[0], psArgs[1:]...)
}
