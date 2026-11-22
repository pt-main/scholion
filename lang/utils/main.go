package utils

import "os"

func ReadFileToString(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func WriteStringToFile(path, data string) error {
	return os.WriteFile(path, []byte(data), 0644)
}
