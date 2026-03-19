package utils

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"
)

// ExtractPDFText extracts text from an io.Reader containing PDF data without using temporary files.
func ExtractPDFText(r io.Reader) (string, error) {
	// ledongthuc/pdf requires io.ReaderAt and the size of the file.
	// Since we are moving away from disk, we read the entire thing into memory once.
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read PDF data: %v", err)
	}

	reader := bytes.NewReader(data)
	contentReader, err := pdf.NewReader(reader, int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %v", err)
	}

	var textBuilder bytes.Buffer
	numPages := contentReader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := contentReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		plainText, err := page.GetPlainText(nil)
		if err != nil {
			return "", fmt.Errorf("failed to extract text from page %d: %v", i, err)
		}

		textBuilder.WriteString(plainText)
		textBuilder.WriteString("\n\n")
	}

	return textBuilder.String(), nil
}
