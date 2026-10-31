package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// renderSecuritySealBanner prints the cryptographic provenance and anti-tamper seal banner.
func renderSecuritySealBanner(w io.Writer, appCtx *AppContext) {
	if appCtx == nil || appCtx.SecuritySeal == nil || appCtx.Flags.Quiet || (appCtx.Flags.Format != "" && appCtx.Flags.Format != "text") {
		return
	}
	seal := appCtx.SecuritySeal
	st := styles(appCtx)

	status := st.Success.Render("SEALED & VERIFIED ✓")
	if seal.InsecureOrigin {
		status = st.Warning.Render(fmt.Sprintf("INSECURE ORIGIN: %s", seal.InsecureReason))
	}

	destHome := ""
	if appCtx.Platform != nil {
		destHome = appCtx.Platform.HomeDir()
	}

	fmt.Fprintf(w, "%s\n", st.Header.Render("── Cryptographic Origin & Provenance Seal ──"))
	fmt.Fprintf(w, "  • Source Origin   : %s\n", seal.ProvenanceOrigin())
	fmt.Fprintf(w, "  • Manifests SHA256: %s\n", seal.ShortManifestDigest())
	if destHome != "" {
		fmt.Fprintf(w, "  • Destination Home: %s\n", destHome)
	}
	fmt.Fprintf(w, "  • Dotfiles SHA256 : %s\n", seal.ShortDotfilesDigest())
	fmt.Fprintf(w, "  • Security Seal   : %s [%s]\n\n", seal.ShortDigest(), status)
}

// verifySecuritySeal performs cryptographic anti-tamper verification against the established seal.
func verifySecuritySeal(cmd *cobra.Command, appCtx *AppContext) error {
	if appCtx == nil || appCtx.SecuritySeal == nil {
		return nil
	}
	return appCtx.SecuritySeal.Verify(cmd.Context())
}
