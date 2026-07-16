package updater

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func VerifyPackage(ctx context.Context, packagePath, teamID string) error {
	if teamID == "" {
		return fmt.Errorf("release build does not contain an expected Apple team ID")
	}
	output, err := exec.CommandContext(ctx, "/usr/sbin/pkgutil", "--check-signature", packagePath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("verify package signature: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if !signatureContainsTeamID(string(output), teamID) {
		return fmt.Errorf("package is not signed by expected Apple team %s", teamID)
	}

	output, err = exec.CommandContext(ctx, "/usr/sbin/spctl", "--assess", "--type", "install", "--verbose=2", packagePath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("verify package trust: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func signatureContainsTeamID(output, teamID string) bool {
	return strings.Contains(output, "("+teamID+")") || strings.Contains(output, "TeamIdentifier="+teamID)
}

func InstallPackage(ctx context.Context, packagePath string) error {
	output, err := exec.CommandContext(ctx, "/usr/sbin/installer", "-pkg", packagePath, "-target", "/").CombinedOutput()
	if err != nil {
		return fmt.Errorf("install package: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
