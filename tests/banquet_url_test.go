package tests

import (
	"testing"

	"github.com/darianmavgo/banquet"
	"github.com/stretchr/testify/assert"
)

func TestBanquetURLUserInfoParsing(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantUser string
		wantHost string
	}{
		{
			name:     "S3 with remote alias",
			url:      "s3://bucket@aws/data/file.csv",
			wantUser: "bucket",
			wantHost: "aws",
		},
		{
			name:     "GCS with remote alias",
			url:      "gs://mybucket@gcloud/data/file.csv",
			wantUser: "mybucket",
			wantHost: "gcloud",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := banquet.ParseBanquet(tt.url)
			assert.NoError(t, err)
			assert.NotNil(t, b.User)
			assert.Equal(t, tt.wantUser, b.User.Username())
			assert.Equal(t, tt.wantHost, b.Host)
		})
	}
}

func TestBanquetColumnPathParsing(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantSelect  []string
		wantOrderBy string
		wantWhere   string
	}{
		{
			name:        "Columns with sort",
			url:         "data.csv;tb0;name,amount;+date",
			wantSelect:  []string{"name", "amount"},
			wantOrderBy: "date",
		},
		{
			name:       "Columns with condition",
			url:        "data.csv;tb0;name,amount;status!=active",
			wantSelect: []string{"name", "amount"},
			wantWhere:  "status != 'active'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := banquet.ParseBanquet(tt.url)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantSelect, b.Select)
			if tt.wantOrderBy != "" {
				assert.Equal(t, tt.wantOrderBy, b.OrderBy)
			}
			if tt.wantWhere != "" {
				assert.Contains(t, b.Where, tt.wantWhere)
			}
		})
	}
}
