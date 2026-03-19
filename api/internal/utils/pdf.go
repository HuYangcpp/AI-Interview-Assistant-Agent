package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/ledongthuc/pdf"
)

// ExtractPDFTextFromFile 从文件路径读取并解析 PDF 文本
func ExtractPDFTextFromFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	reader := bytes.NewReader(data)
	contentReader, err := pdf.NewReader(reader, int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("创建PDF reader失败: %w", err)
	}

	var sb bytes.Buffer
	for i := 1; i <= contentReader.NumPage(); i++ {
		page := contentReader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
