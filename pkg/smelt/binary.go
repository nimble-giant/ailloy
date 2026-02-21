package smelt

import "fmt"

// PackageBinary packages a mold into a self-contained binary by rendering
// blanks, compressing the output, and appending it to the ailloy binary
// using the stuffbin pattern. This is not yet implemented.
func PackageBinary(moldDir, outputDir string) (string, int64, error) {
	return "", 0, fmt.Errorf("binary output is not yet implemented (see: github.com/knadh/stuffbin)")
}
