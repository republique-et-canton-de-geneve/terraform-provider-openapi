package spec

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/pb33f/libopenapi"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// LoadModel fetches and parses an OpenAPI 3.x spec from a file path or HTTP(S) URL.
func LoadModel(source string) (*libopenapi.DocumentModel[v3high.Document], error) {
	data, err := readSource(source)
	if err != nil {
		return nil, err
	}
	doc, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}
	model, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		return nil, fmt.Errorf("building v3 model: %w", errs[0])
	}
	return model, nil
}

// readSource reads raw bytes from a local file path or an HTTP(S) URL.
func readSource(source string) ([]byte, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		resp, err := http.Get(source) //nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("fetching spec from %s: %w", source, err)
		}
		defer func() { _ = resp.Body.Close() }()
		return io.ReadAll(resp.Body)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("reading spec file %s: %w", source, err)
	}
	return data, nil
}
