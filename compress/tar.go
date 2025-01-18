package compress

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Build(dirToTar string, output string) error {
	// 创建 TAR.GZ 文件
	tarFile, err := os.Create(output)
	if err != nil {
		fmt.Println("Error creating TAR.GZ file:", err)
		return err
	}
	defer tarFile.Close()

	// 创建 Gzip Writer
	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()

	// 创建 TAR Writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// 遍历当前目录的所有文件和子目录
	return filepath.Walk(dirToTar, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 如果是目录，跳过
		if info.IsDir() {
			return nil
		}

		// 打开要压缩的文件
		fileToTar, err := os.Open(file)
		if err != nil {
			return err
		}
		defer fileToTar.Close()

		// 获取相对路径
		relativePath, err := filepath.Rel(dirToTar, file)
		if err != nil {
			return err
		}

		// 创建 TAR 文件中的文件头
		header, err := tar.FileInfoHeader(info, relativePath)
		if err != nil {
			return err
		}
		header.Name = relativePath

		// 将文件头写入 TAR 文件
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// 将文件内容写入 TAR 文件
		_, err = io.Copy(tarWriter, fileToTar)
		if err != nil {
			return err
		}

		return nil
	})
}
