package application

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain"
)

func benchmarkSearchReport() domain.Report {
	source := strings.Repeat("if value != nil { return value }\n", 50)
	report := domain.Report{Files: make([]domain.File, 100)}
	for fileIndex := range report.Files {
		file := &report.Files[fileIndex]
		file.Path = fmt.Sprintf("file%d.go", fileIndex)
		for index := range 100 {
			file.Functions = append(file.Functions, domain.Function{
				Name: fmt.Sprintf("Function%d", index), Complexity: index%5 + 1,
				Line: index*50 + 1, Source: source,
			})
			file.Total += index%5 + 1
		}
	}
	return rankReport(report)
}

func BenchmarkControlSearch(b *testing.B) {
	model := Model{report: benchmarkSearchReport()}
	command := controlCommand{action: searchAction, search: SearchQuery{Limit: 20}, reply: make(chan controlReply, 1)}
	b.ResetTimer()
	for b.Loop() {
		model.updateControl(command)
		if reply := <-command.reply; reply.err != nil {
			b.Fatal(reply.err)
		}
	}
}
