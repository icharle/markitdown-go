package markitdown

import "errors"

// Convert 调用 markitdown 转换文件为 Markdown
func Convert(filePath string) (string, error) {
	// 检查文件路径
	if filePath == "" {
		return "", errors.New("file path cannot be empty")
	}

	// 调用 Python C API
	result, err := callMarkItDown(filePath)
	if err != nil {
		return "", err
	}

	return result, nil
}
