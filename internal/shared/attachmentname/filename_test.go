package attachmentname

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSafeDownloadFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "keeps simple filename", filename: "report.pdf", want: "report.pdf"},
		{name: "trims spaces", filename: "  report.pdf  ", want: "report.pdf"},
		{name: "drops parent directories", filename: "../secret/report.pdf", want: "report.pdf"},
		{name: "normalizes windows path", filename: `..\\secret\\report.pdf`, want: "report.pdf"},
		{name: "drops control characters", filename: "report\n.pdf", want: "report.pdf"},
		{name: "empty fallback", filename: "   ", want: "attachment"},
		{name: "dot fallback", filename: ".", want: "attachment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Normalize(tt.filename))
		})
	}
}
