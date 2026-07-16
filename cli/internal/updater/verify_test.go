package updater

import "testing"

func TestSignatureContainsTeamID(t *testing.T) {
	t.Parallel()
	if !signatureContainsTeamID("Developer ID Installer: Example Corp (ABC123XYZ)", "ABC123XYZ") {
		t.Fatal("expected team ID to be recognized")
	}
	if signatureContainsTeamID("Developer ID Installer: Other Corp (OTHERTEAM)", "ABC123XYZ") {
		t.Fatal("unexpected signer accepted")
	}
}
