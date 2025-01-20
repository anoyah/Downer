package tools

import "errors"

var (
	// 文件不存在
	ErrFileExist = errors.New("the file already exists, please rename output file")
)
