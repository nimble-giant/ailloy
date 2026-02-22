package foundry

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// extractTar extracts tar data into the given directory.
func extractTar(data []byte, destDir string) error {
	tr := tar.NewReader(bytes.NewReader(data))

	absDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("resolving dest dir: %w", err)
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		target := filepath.Join(absDir, filepath.Clean(hdr.Name))
		if !strings.HasPrefix(target, absDir+string(filepath.Separator)) && target != absDir {
			return fmt.Errorf("tar entry %q would escape destination", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0750); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0750); err != nil {
				return fmt.Errorf("creating parent dir for %s: %w", target, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0644) //#nosec G115,G304 -- target is validated against traversal above
			if err != nil {
				return fmt.Errorf("creating file %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil { //#nosec G110 -- tar from local bare clone
				_ = f.Close()
				return fmt.Errorf("writing file %s: %w", target, err)
			}
			_ = f.Close()
		}
	}
	return nil
}
