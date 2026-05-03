package commands

import (
	"context"
	"io/fs"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

func init() {
	mold.ResolveSchemaFunc = func(ctx context.Context, source string) (fs.FS, func(), error) {
		fsys, _, err := foundry.ResolveWithMetadata(source)
		if err != nil {
			return nil, nil, err
		}
		return fsys, nil, nil
	}
}
