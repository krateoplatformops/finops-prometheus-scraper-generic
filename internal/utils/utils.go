package utils

import (
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type ContextKey string

const CertFilePathKey ContextKey = "certFilePath"

func Fatal(err error) {
	if err != nil {
		log.Logger.Warn().Err(err).Msg("An error has occured, continuing...")
		time.Sleep(5 * time.Second)
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
