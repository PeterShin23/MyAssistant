package openai

import "fmt"

// import (
// 	"context"
// 	"fmt"
// 	"os"
// 	"path/filepath"

// 	"github.com/openai/openai-go/openai"
// )

func Process(screenshotPath string, audioPath string) error {
	var log = fmt.Sprintf("Now in processing step, %s, %s", screenshotPath, audioPath)

	fmt.Println(log)

	return nil
}
