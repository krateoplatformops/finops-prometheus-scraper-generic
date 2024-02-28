package utils

import (
	"fmt"
	"strings"
)

func Fatal(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func ConvertIntoRowWithMaximumLength10k(allRows []string, byteSizeLimit int) []string {
	result := []string{}
	temp := ""
	for _, row := range allRows {
		if len(temp)+len(row) > byteSizeLimit {
			result = append(result, strings.Trim(temp, "\n"))
			temp = ""
		}
		temp += row + "\n"
	}
	if temp != "" {
		return append(result, strings.Trim(temp, "\n"))
	}
	return result
}
